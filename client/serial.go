package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kjx98/crc16"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/test"
	"go.bug.st/serial"
)

// SerialDev represents a serial (MCU) config
type SerialDev struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Port        string `point:"port"`
	Baud        string `point:"baud"`
	// Protocol selects the wire format: "binary" (default, COBS framed) or
	// "shell" (Zephyr console shell, ASCII lines).
	Protocol string `point:"protocol"`
	// Timeout is the seconds without any traffic before the shell protocol
	// declares the link disconnected. 0 means defaultShellTimeout.
	Timeout int `point:"timeout"`
	// LogConsole mirrors every line received from the MCU to the server log.
	// Shell protocol only.
	LogConsole        bool   `point:"logConsole"`
	MaxMessageLength  int    `point:"maxMessageLength"`
	HRDestNode        string `point:"hrDest"`
	SyncParent        bool   `point:"syncParent"`
	Debug             int    `point:"debug"`
	Disabled          bool   `point:"disabled"`
	Log               string `point:"log"`
	Rx                int    `point:"rx"`
	RxReset           bool   `point:"rxReset"`
	Tx                int    `point:"tx"`
	TxReset           bool   `point:"txReset"`
	HrRx              int64  `point:"hrRx"`
	HrRxReset         bool   `point:"hrRxReset"`
	Uptime            int    `point:"uptime"`
	ErrorCount        int    `point:"errorCount"`
	ErrorCountReset   bool   `point:"errorCountReset"`
	ErrorCountHR      int    `point:"errorCountHR"`
	ErrorCountResetHR bool   `point:"errorCountResetHR"`
	Rate              bool   `point:"rate"`
	RateHR            bool   `point:"rate"`
	Connected         bool   `point:"connected"`
	Download          string `point:"download"`
	Progress          int    `point:"progress"`
	Files             []File `child:"file"`
}

type sendData struct {
	seq     byte
	ack     bool
	subject string
	points  data.Points
}

// SerialDevClient is a SIOT client used to manage serial devices
type SerialDevClient struct {
	nc                  *nats.Conn
	config              SerialDev
	stop                chan struct{}
	newPoints           chan NewPoints
	newEdgePoints       chan NewPoints
	wrSeq               byte // convention is we increment this right before sending a packet
	lastSendStats       time.Time
	natsSub             string
	natsSubSerialPoints string
	natsSubHRUp         string
	parentSubscription  *nats.Subscription
	ratePointCount      int
	ratePointCountHR    int
	rateLastSend        time.Time
	port                serialPort
	sendPointsCh        chan sendData
	// echoCache records points written to the MCU in shell mode so the
	// echo the MCU sends back can be recognized and dropped.
	echoCache map[echoKey]echoEntry
	// lastRx is when we last received anything from the MCU, used by the
	// shell mode connection watchdog.
	lastRx time.Time
}

// serialPort is the framing layer between the raw port and the client. Read
// returns exactly one message per call: a COBS frame for the binary protocol,
// or one console line for the shell protocol.
type serialPort interface {
	io.ReadWriteCloser
	SetDebug(int)
}

// shell returns true if this client is configured for the Zephyr shell
// protocol. An empty Protocol means binary, so existing nodes are unaffected.
func (sd *SerialDevClient) shell() bool {
	return sd.config.Protocol == data.PointValueProtocolShell
}

// NewSerialDevClient ...
func NewSerialDevClient(nc *nats.Conn, config SerialDev) Client {
	ret := &SerialDevClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		natsSub:       SubjectNodePoints(config.ID),
		sendPointsCh:  make(chan sendData),
	}

	ret.populateNatsSubjects()

	return ret
}

func (sd *SerialDevClient) populateNatsSubjects() {
	phrup := fmt.Sprintf("phrup.%v.%v", sd.config.Parent, sd.config.ID)
	if sd.config.HRDestNode != "" {
		phrup = fmt.Sprintf("phrup.%v.%v", sd.config.HRDestNode, sd.config.ID)
	}
	sd.natsSubHRUp = phrup

	if sd.parentSubscription != nil {
		err := sd.parentSubscription.Unsubscribe()
		if err != nil {
			log.Println("Serial: error unsubscribing from parent sub:", err)
		}
		sd.parentSubscription = nil
	}

	if sd.config.SyncParent {
		sd.natsSubSerialPoints = SubjectNodePoints(sd.config.Parent)
		var err error
		// Copy some config to avoid race conditions
		serialID := sd.config.ID
		debug := sd.config.Debug
		sd.parentSubscription, err = sd.nc.Subscribe(sd.natsSubSerialPoints+".>", func(msg *nats.Msg) {
			points, err := data.DecodePoints(msg.Data)
			if err != nil {
				log.Println("Error decoding points in serial parent:", err)
				return
			}

			// only send points whose origin is not the serial node ID as those are just
			// getting echo'd back
			var pointsToSend data.Points

			for _, p := range points {
				if p.Origin != serialID {
					pointsToSend = append(pointsToSend, p)
				}
			}

			if len(pointsToSend) > 0 {
				if sd.port == nil {
					if debug >= 4 {
						log.Printf("Serial port closed; points not sent: %v", pointsToSend)
					}
					return
				}
				sd.sendPointsCh <- sendData{points: pointsToSend}
			}
		})
		if err != nil {
			log.Println("Error subscribing to serial parent:", err)
		}
	} else {
		sd.natsSubSerialPoints = SubjectNodePoints(sd.config.ID)
	}
}

// if seq == 0, then sd.wrSeq is used
func (sd *SerialDevClient) sendPointsToDevice(seq byte, ack bool, sub string, pts data.Points) error {
	if seq == 0 {
		seq = sd.wrSeq
	}

	if sub == "" {
		sub = "proto"
	}

	d, err := SerialEncode(seq, sub, pts)
	if err != nil {
		return fmt.Errorf("error encoding points to send to MCU: %w", err)
	}

	if sd.config.Debug >= 4 {
		if len(pts) > 0 {
			log.Printf("SER TX (%v) seq:%v sub:%v :\n%v", sd.config.Description, seq, sub, pts)
		} else {
			log.Printf("SER TX (%v) seq:%v sub:%v\n", sd.config.Description, seq, sub)
		}
	}

	if sd.port != nil {
		_, err = sd.port.Write(d)
		if err != nil {
			return fmt.Errorf("error writing data to port: %w", err)
		}

		sd.config.Tx++
		err = SendPoints(sd.nc, sd.natsSub,
			data.Points{data.NewPointFloat(data.PointTypeTx, "", float64(sd.config.Tx))},
			false)

		if err != nil {
			return fmt.Errorf("error sending Serial tx stats: %w", err)
		}
	}

	if !ack {
		_ = ack
		// TODO: we need to check for response and implement retries
		// yet.
	}

	return nil
}

// publishReadResults sends points decoded from the MCU upstream, along with
// the periodic rx/rate statistics. Shared by both protocols: everything from
// here on is independent of how the bytes were framed.
func (sd *SerialDevClient) publishReadResults(points, adminPoints data.Points) {
	if time.Since(sd.lastSendStats) > time.Second*5 {
		adminPoints = append(adminPoints,
			data.Points{
				data.NewPointFloat(data.PointTypeRx, "", float64(sd.config.Rx)),
				data.NewPointFloat(data.PointTypeHrRx, "", float64(sd.config.HrRx)),
			}...)
		sd.lastSendStats = time.Now()
	}

	if time.Since(sd.rateLastSend) > time.Second {
		now := time.Now()
		elapsedSec := now.Sub(sd.rateLastSend).Seconds()
		rate := float64(sd.ratePointCount) / elapsedSec
		rateHR := float64(sd.ratePointCountHR) / elapsedSec
		adminPoints = append(adminPoints,
			data.NewPointFloat(data.PointTypeRate, "", rate),
			data.NewPointFloat(data.PointTypeRateHR, "", rateHR),
		)
		sd.rateLastSend = now
		sd.ratePointCount = 0
		sd.ratePointCountHR = 0
	}

	if sd.config.SyncParent {
		// add serial ID to origin for all points we send to the parent
		for i := range points {
			points[i].Origin = sd.config.ID
		}
	}

	if len(points) > 0 {
		err := SendPoints(sd.nc, sd.natsSubSerialPoints, points, false)
		if err != nil {
			log.Println("Error sending points received from MCU:", err)
		}
	}

	if len(adminPoints) > 0 {
		err := SendPoints(sd.nc, sd.natsSub, adminPoints, false)
		if err != nil {
			log.Println("Error sending admin points:", err)
		}
	}
}

type downloadState struct {
	name         string
	fileBuf      *bytes.Buffer
	packet       *bytes.Buffer
	seq          byte
	currentBlock int
}

// Run the main logic for this client and blocks until stopped
func (sd *SerialDevClient) Run() error {
	log.Println("Starting serial client:", sd.config.Description)

	// -1 indicates nothing is downloading right now
	dlTimeout := time.NewTicker(10 * time.Second)
	dlTimeout.Stop()

	if sd.config.Connected {
		sd.config.Connected = false
		err := SendNodePoint(sd.nc, sd.config.ID, data.NewPointFloat(data.PointTypeConnected, "", 0), false)
		if err != nil {
			log.Println("Error sending connected point")
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("error creating a new fsnotify watcher: %w", err)

	}
	defer watcher.Close()

	if sd.config.Port != "" {
		err := watcher.Add(filepath.Dir(sd.config.Port))
		if err != nil {
			log.Println("Error adding watcher for:", sd.config.Port)
		}
	}

	checkPortDur := time.Second * 10
	timerCheckPort := time.NewTimer(checkPortDur)

	// Shell mode watchdog: the port staying open says nothing about whether
	// the MCU is alive, so track when we last heard anything.
	shellWatchdog := time.NewTicker(time.Second * 5)
	defer shellWatchdog.Stop()

	serialReadData := make(chan []byte)
	listenerClosed := make(chan struct{})
	listenerSerialErr := make(chan struct{})

	dlState := downloadState{}

	dlFileStop := func() {
		dlState.name = ""
		_ = SendNodePoint(sd.nc, sd.config.ID, data.Point{Type: data.PointTypeDownload}, true)
	}

	dlFileSend := func() {
		if dlState.name == "" {
			log.Println("File download error, no file queued")
			return
		}

		// packet format
		// sequence (1 byte)
		// subject (16 bytes, always file)
		// filename (16 bytes)
		// block index (4 bytes)
		// data
		// crc (2 bytes)

		// we save off packet in struct so we can resend on retries
		dlState.packet = &bytes.Buffer{}

		sd.wrSeq++
		dlState.seq = sd.wrSeq
		err := dlState.packet.WriteByte(sd.wrSeq)
		if err != nil {
			log.Println("Error writing seq to file packet: ", err)
			dlFileStop()
			return
		}

		// subject is always file
		f := make([]byte, 16)
		copy(f, []byte("file"))
		_, err = dlState.packet.Write(f)
		if err != nil {
			log.Println("Error writing file to file packet: ", err)
			dlFileStop()
			return
		}

		n := make([]byte, 16)
		copy(n, []byte(dlState.name))
		_, err = dlState.packet.Write(n)
		if err != nil {
			log.Println("Error writing filename to file packet: ", err)
			dlFileStop()
			return
		}

		if dlState.fileBuf.Len() <= 0 {
			dlState.currentBlock = -1
		}

		// copy block number
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(dlState.currentBlock))
		_, err = dlState.packet.Write(b)
		if err != nil {
			log.Println("Error writing block number to packet: ", err)
			dlFileStop()
			return
		}

		// copy payload
		_, err = io.CopyN(dlState.packet, dlState.fileBuf, 400)
		if err != nil {
			if err != io.EOF {
				log.Println("Error copying file packet content: ", err)
				dlFileStop()
				return
			}
		}

		crc := crc16.ChecksumCCITT(dlState.packet.Bytes())
		_ = binary.Write(dlState.packet, binary.LittleEndian, crc)

		if sd.config.Debug >= 4 {
			log.Printf("SER TX file seq:%v name:%v block:%v len:%v", sd.wrSeq, dlState.name, dlState.currentBlock, dlState.packet.Len())
		}

		if sd.port != nil {
			// _, err = io.Copy(sd.port, dlState.packet)
			_, err = sd.port.Write(dlState.packet.Bytes())

			if err != nil {
				log.Println("Error writing file to cobs wrapper: ", err)
				return
			}
		}
	}

	// dlRetry := func() {

	// }

	dlFileStart := func(name string) {
		if dlState.name != "" {
			log.Println("Error, file download already in progress")
			return
		}

		if len(name) > 16 {
			log.Println("file name is too long: ", name)
			return
		}

		// see if we can find the file
		for _, f := range sd.config.Files {
			if f.Name == name {
				d, err := f.GetContents()
				if err != nil {
					log.Println("Error decoding file: ", err)
					continue
				}
				dlState.fileBuf = bytes.NewBuffer(d)
				dlState.name = name
				dlState.currentBlock = 0

				dlFileSend()
				return
			}
		}

		log.Println("Error could not find file: ", name)
	}

	closePort := func() {
		if sd.port != nil {
			log.Println("Closing serial port:", sd.config.Description)
			sd.port.Close()
		}
		sd.port = nil

		sd.config.Connected = false
		err := SendNodePoint(sd.nc, sd.config.ID, data.NewPointFloat(data.PointTypeConnected, "", 0), false)
		if err != nil {
			log.Println("Error sending connected point")
		}
	}

	if sd.config.Download != "" {
		_ = SendNodePoint(sd.nc, sd.config.ID, data.Point{Type: data.PointTypeDownload}, true)
	}

	listener := func(port io.ReadWriteCloser, maxMessageLen int) {
		errCount := 0
		for {
			buf := make([]byte, maxMessageLen)
			c, err := port.Read(buf)
			if err != nil {
				if err != io.EOF && err.Error() != "Port has been closed" {
					log.Printf("Error reading port %v: %v\n", sd.config.Description, err)

					listenerSerialErr <- struct{}{}

					// we don't want to reset the port on every COBS
					// decode error, so accumulate a few before we do this
					if err == ErrCobsDecodeError ||
						err == ErrCobsTooMuchData {
						errCount++
						if errCount < 10000000 {
							continue
						}
					}

					listenerClosed <- struct{}{}
					return
				}
			}
			if c <= 0 {
				continue
			}

			errCount = 0

			buf = buf[0:c]
			serialReadData <- buf
		}
	}

	openPort := func() {
		if sd.config.MaxMessageLength <= 0 {
			sd.config.MaxMessageLength = 1024
			err := SendPoints(sd.nc, sd.natsSub,
				data.Points{data.NewPointFloat(data.PointTypeMaxMessageLength, "", 1024)}, true)
			if err != nil {
				log.Println("Error sending max message len message:", err)
			}
		}

		// make sure port is closed before we try to (re)open it
		closePort()

		if sd.config.Disabled {
			closePort()
			timerCheckPort.Stop()
			return
		}

		var io io.ReadWriteCloser

		if sd.config.Port == "serialfifo" {
			// we are in test mode and using unix fifos instead of
			// real serial ports. The fifo must already by started
			// by the test harness
			var err error
			io, err = test.NewFifoB(sd.config.Port)
			if err != nil {
				log.Println("SerialDevClient: error opening fifo:", err)
				return
			}
		} else {
			if sd.config.Port == "" || sd.config.Baud == "" {
				log.Printf("Serial port %v not configured\n", sd.config.Description)
				timerCheckPort.Reset(checkPortDur)
				return
			}

			baud, err := strconv.Atoi(sd.config.Baud)

			if err != nil {
				log.Printf("Serial port %v invalid baud\n", sd.config.Description)
				timerCheckPort.Reset(checkPortDur)
				return
			}

			mode := &serial.Mode{
				BaudRate: baud,
			}

			sp, err := serial.Open(sd.config.Port, mode)
			if err != nil {
				log.Printf("Error opening serial port %v: %v", sd.config.Description,
					err)
				timerCheckPort.Reset(checkPortDur)
				return
			}

			time.Sleep(time.Millisecond)
			err = sp.SetDTR(false)
			if err != nil {
				log.Printf("Error clearing serial port DTR: %v\n", err)
			}

			time.Sleep(time.Millisecond * 100)
			err = sp.SetDTR(true)
			if err != nil {
				log.Printf("Error setting serial port DTR: %v\n", err)
			}

			io = sp
		}

		if sd.shell() {
			sd.port = NewLineWrapper(io, sd.config.MaxMessageLength)
		} else {
			sd.port = NewCobsWrapper(io, sd.config.MaxMessageLength)
		}
		sd.port.SetDebug(sd.config.Debug)
		timerCheckPort.Stop()

		log.Println("Serial port opened:", sd.config.Description)

		go listener(sd.port, sd.config.MaxMessageLength)

		if sd.shell() {
			// The shell protocol has no time sync packet, and being able to
			// open the port says nothing about whether anything is alive on
			// the other end. connected is set when the first line arrives.
			sd.echoCache = map[echoKey]echoEntry{}
			sd.lastRx = time.Time{}
			sd.sendShellHandshake()
			return
		}

		p := data.Points{{
			Time: time.Now(),
			Type: data.PointTypeTimeSync,
		}}

		sd.config.Connected = true
		err := SendNodePoint(sd.nc, sd.config.ID, data.NewPointFloat(data.PointTypeConnected, "", 1), false)
		if err != nil {
			log.Println("Error sending connected point")
		}

		sd.wrSeq++
		err = sd.sendPointsToDevice(sd.wrSeq, false, "", p)
		if err != nil {
			log.Println("Error sending time sync point to device: %w", err)
		}
	}

	openPort()

exitSerialClient:

	for {
		select {
		case <-sd.stop:
			break exitSerialClient
		case <-timerCheckPort.C:
			openPort()
		case <-shellWatchdog.C:
			if !sd.shell() || sd.port == nil {
				break
			}

			sd.expireEchoCache()

			// lastRx is zero until the first line arrives, so a board that
			// never says anything is reported disconnected rather than
			// staying optimistically connected from the port open.
			silent := time.Since(sd.lastRx)
			if sd.lastRx.IsZero() {
				silent = sd.shellTimeout() + time.Second
			}

			if sd.config.Connected && silent > sd.shellTimeout() {
				log.Printf("Serial %v: no data for %v, marking disconnected",
					sd.config.Description, silent.Round(time.Second))
				sd.config.Connected = false
				err := SendNodePoint(sd.nc, sd.config.ID,
					data.NewPointFloat(data.PointTypeConnected, "", 0), false)
				if err != nil {
					log.Println("Error sending connected point:", err)
				}
			}
		case <-listenerClosed:
			closePort()
			timerCheckPort.Reset(checkPortDur)
		case <-listenerSerialErr:
			sd.config.ErrorCount++
			points := []data.Point{data.NewPointFloat(data.PointTypeErrorCount, "", float64(sd.config.ErrorCount))}
			err := SendPoints(sd.nc, sd.natsSub, points, false)
			if err != nil {
				log.Println("Error sending error points:", err)
			}
		case e, ok := <-watcher.Events:
			if ok {
				if e.Name == sd.config.Port {
					switch e.Op {
					case fsnotify.Remove:
						closePort()
					case fsnotify.Create:
						openPort()
					}
				}
			}
		case rd := <-serialReadData:
			sd.lastRx = time.Now()

			if sd.shell() {
				// Any line is evidence the far end is alive. The shell
				// protocol has no framing beyond the line itself, so
				// decoding is only classification.
				if !sd.config.Connected {
					sd.config.Connected = true
					err := SendNodePoint(sd.nc, sd.config.ID,
						data.NewPointFloat(data.PointTypeConnected, "", 1), false)
					if err != nil {
						log.Println("Error sending connected point:", err)
					}
				}

				shellPoints, shellAdmin := sd.handleShellLine(string(rd))
				sd.config.Rx++
				sd.ratePointCount += len(shellPoints)

				if !sd.config.SyncParent && len(shellPoints) > 0 {
					err := data.MergePoints(sd.config.ID, shellPoints, &sd.config)
					if err != nil {
						log.Println("error merging new points:", err)
					}
				}

				sd.publishReadResults(shellPoints, shellAdmin)
				break
			}

			if sd.config.Debug >= 8 {
				log.Println("SER RX RAW:", test.HexDump(rd))
			}

			// decode serial packet
			seq, subject, payload, err := SerialDecode(rd)
			if err != nil {
				log.Printf("Serial framing error (sub:%v): %v", subject, err)

				var t string
				var cnt int

				if subject == "phr" {
					t = data.PointTypeErrorCountHR
					sd.config.ErrorCountHR++
					cnt = sd.config.ErrorCountHR
				} else {
					t = data.PointTypeErrorCount
					sd.config.ErrorCount++
					cnt = sd.config.ErrorCount
				}

				err := SendPoints(sd.nc, sd.natsSub, []data.Point{data.NewPointFloat(t, "", float64(cnt))}, false)
				if err != nil {
					log.Println("Error sending error points:", err)
				}

				break
			}

			if subject == "ack" {
				if sd.config.Debug >= 4 {
					log.Printf("SER RX (%v) seq:%v sub:%v", sd.config.Description, seq, subject)
				}
				// TODO we need to handle acks, retries, etc
				if dlState.name != "" {
					if seq == dlState.seq {
						if dlState.currentBlock == -1 {
							log.Println("File download finished: ", dlState.name)
							dlFileStop()
						} else {
							dlState.currentBlock++
							dlFileSend()
						}
					}
				}
				break
			}

			if subject == "phr" {
				// we have high rate points
				sd.config.HrRx++
				err := sd.nc.Publish(sd.natsSubHRUp, payload)
				if err != nil {
					log.Println("Error publishing HR data:", err)
				}
				sd.ratePointCountHR++
				// we're done
				break
			}

			if subject == "log" {
				points := data.Points{data.NewPointString(data.PointTypeLog, "", string(payload))}

				if sd.config.Debug >= 1 {
					log.Printf("Serial client %v: log: %v\n",
						sd.config.Description, string(payload))
				}
				err := SendPoints(sd.nc, sd.natsSubSerialPoints, points, false)
				if err != nil {
					log.Println("Error sending log point from MCU:", err)
				}
			}

			// decode binary payload
			points, errDecode := data.DecodePoints(payload)
			var adminPoints data.Points

			sd.config.Rx++

			// make sure time is set on all points
			for i, p := range points {
				if p.Time.Year() <= 1980 {
					points[i].Time = time.Now()
				}
			}

			sd.ratePointCount += len(points)

			if errDecode == nil && len(points) > 0 {
				if sd.config.Debug >= 4 {
					log.Printf("SER RX (%v) seq:%v sub:%v\n%v", sd.config.Description, seq, subject, points)
				}

				// send response
				err := sd.sendPointsToDevice(seq, false, "ack", nil)
				if err != nil {
					log.Println("Error sending ack to device:", err)
				}

				if !sd.config.SyncParent {
					err = data.MergePoints(sd.config.ID, points, &sd.config)
					if err != nil {
						log.Println("error merging new points:", err)
					}
				}
			} else {
				log.Println("Error decoding serial packet from device:",
					sd.config.Description, errDecode)
				sd.config.ErrorCount++
				adminPoints = append(adminPoints,
					data.NewPointFloat(data.PointTypeErrorCount, "", float64(sd.config.ErrorCount)))
			}

			sd.publishReadResults(points, adminPoints)

		case pts := <-sd.newPoints:
			op := false
			updateNatsSubjects := false
			for _, p := range pts.Points {
				// check if any of the config changes should cause us to re-open the port
				if p.Type == data.PointTypePort ||
					p.Type == data.PointTypeBaud ||
					p.Type == data.PointTypeDisabled ||
					p.Type == data.PointTypeProtocol ||
					p.Type == data.PointTypeMaxMessageLength {
					op = true
				}

				if p.Type == data.PointTypePort {
					err := watcher.Add(filepath.Dir(p.Txt()))
					if err != nil {
						log.Println("Error adding watcher on serial port name change:", p.Txt())
					}
				}

				if p.Type == data.PointTypeDisabled {
					if p.Val() == 0 {
						closePort()
					} else {
						op = true
					}
				}

				if p.Type == data.PointTypeHRDest {
					updateNatsSubjects = true
				}

				if p.Type == data.PointTypeSyncParent {
					updateNatsSubjects = true
				}

				// guard against the port being closed: a debug point can
				// arrive at any time, including before the port opens
				if p.Type == data.PointTypeDebug && sd.port != nil {
					sd.port.SetDebug(int(p.Val()))
				}

				if p.Type == data.PointTypeDownload {
					if p.Txt() == "" {
						log.Println("Stopping download: ", dlState.name)
						dlFileStop()
					} else {
						dlFileStart(p.Txt())
						log.Println("Starting download: ", p.Txt())
					}
				}
			}

			err := data.MergePoints(pts.ID, pts.Points, &sd.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}

			if updateNatsSubjects {
				sd.populateNatsSubjects()
			}

			if op {
				openPort()
			}

			if sd.port == nil {
				break
			}

			if sd.config.ErrorCountReset {
				points := data.Points{
					data.NewPointFloat(data.PointTypeErrorCount, "", 0),
					data.NewPointFloat(data.PointTypeErrorCountReset, "", 0),
				}
				err = SendPoints(sd.nc, sd.natsSub, points, false)
				if err != nil {
					log.Println("Error resetting MCU error count:", err)
				}

				sd.config.ErrorCountReset = false
				sd.config.ErrorCount = 0
			}

			if sd.config.ErrorCountResetHR {
				points := data.Points{
					data.NewPointFloat(data.PointTypeErrorCountHR, "", 0),
					data.NewPointFloat(data.PointTypeErrorCountResetHR, "", 0),
				}
				err = SendPoints(sd.nc, sd.natsSub, points, false)
				if err != nil {
					log.Println("Error resetting MCU error count:", err)
				}

				sd.config.ErrorCountResetHR = false
				sd.config.ErrorCountHR = 0
			}

			if sd.config.RxReset {
				points := data.Points{
					data.NewPointFloat(data.PointTypeRx, "", 0),
					data.NewPointFloat(data.PointTypeRxReset, "", 0),
				}
				err = SendPoints(sd.nc, sd.natsSub, points, false)
				if err != nil {
					log.Println("Error resetting MCU error count:", err)
				}

				sd.config.RxReset = false
				sd.config.Rx = 0
			}

			if sd.config.HrRxReset {
				points := data.Points{
					data.NewPointFloat(data.PointTypeHrRx, "", 0),
					data.NewPointFloat(data.PointTypeHrRxReset, "", 0),
				}
				err = SendPoints(sd.nc, sd.natsSub, points, false)
				if err != nil {
					log.Println("Error resetting MCU error count:", err)
				}

				sd.config.HrRxReset = false
				sd.config.HrRx = 0
			}

			if sd.config.TxReset {
				points := data.Points{
					data.NewPointFloat(data.PointTypeTx, "", 0),
					data.NewPointFloat(data.PointTypeTxReset, "", 0),
				}
				err = SendPoints(sd.nc, sd.natsSub, points, false)
				if err != nil {
					log.Println("Error resetting MCU error count:", err)
				}

				sd.config.TxReset = false
				sd.config.Tx = 0
			}

			// check if we have any points that need sent to MCU
			if !sd.config.SyncParent {
				toSend := data.Points{}
				for _, p := range pts.Points {
					switch p.Type {
					case data.PointTypePort,
						data.PointTypeBaud,
						data.PointTypeDescription,
						data.PointTypeErrorCount,
						data.PointTypeErrorCountReset,
						data.PointTypeRxReset,
						data.PointTypeTxReset,
						// SIOT-side link configuration; the MCU has no use
						// for these and should not be told about them
						data.PointTypeProtocol,
						data.PointTypeTimeout,
						data.PointTypeLogConsole:
						continue
					}

					// strip off Origin as MCU does not need that
					p.Origin = ""
					toSend = append(toSend, p)
				}

				if len(toSend) > 0 {
					if sd.shell() {
						err := sd.sendPointsToDeviceShell(toSend)
						if err != nil {
							log.Println("Error sending points to serial device:", err)
						}
					} else {
						sd.wrSeq++
						err := sd.sendPointsToDevice(sd.wrSeq, false, "", toSend)
						if err != nil {
							log.Println("Error sending points to serial device:", err)
						}
					}
				}
			}

		case pts := <-sd.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &sd.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}

		case sData := <-sd.sendPointsCh:
			if sd.shell() {
				err := sd.sendPointsToDeviceShell(sData.points)
				if err != nil {
					log.Println("Error sending data to device: ", err)
				}
				break
			}

			err := sd.sendPointsToDevice(sData.seq, sData.ack, sData.subject, sData.points)
			if err != nil {
				log.Println("Error sending data to device: ", err)
			}

			// TODO need to send edge points to MCU, not implemented yet
		}
	}

	log.Println("Stopping serial client:", sd.config.Description)
	closePort()
	if sd.parentSubscription != nil {
		err := sd.parentSubscription.Unsubscribe()
		if err != nil {
			log.Println("Error unsubscribing:", err)
		}
	}

	return nil

}

// Stop sends a signal to the Run function to exit
func (sd *SerialDevClient) Stop(_ error) {
	close(sd.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (sd *SerialDevClient) Points(nodeID string, points []data.Point) {
	sd.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (sd *SerialDevClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	sd.newEdgePoints <- NewPoints{nodeID, parentID, points}
}
