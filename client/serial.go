package client

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/test"
	"go.bug.st/serial"
)

// SerialDev represents a serial (MCU) config
type SerialDev struct {
	ID                string `node:"id"`
	Parent            string `node:"parent"`
	Description       string `point:"description"`
	Port              string `point:"port"`
	Baud              string `point:"baud"`
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
	wrSeq               byte
	lastSendStats       time.Time
	natsSub             string
	natsSubSerialPoints string
	natsSubHRUp         string
	parentSubscription  *nats.Subscription
	ratePointCount      int
	ratePointCountHR    int
	rateLastSend        time.Time
	portCobsWrapper     *CobsWrapper
	sendPointsCh        chan sendData
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
		phrup = fmt.Sprintf("phrup.%v.x", sd.config.HRDestNode)
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
		sd.parentSubscription, err = sd.nc.Subscribe(sd.natsSubSerialPoints, func(msg *nats.Msg) {
			points, err := data.PbDecodePoints(msg.Data)
			if err != nil {
				log.Println("Error decoding points in serial parent:", err)
				return
			}

			// only send points whose orgin is not the serial node ID as those are just
			// getting echo'd back
			var pointsToSend data.Points

			for _, p := range points {
				if p.Origin != serialID {
					pointsToSend = append(pointsToSend, p)
				}
			}

			if len(pointsToSend) > 0 {
				if sd.portCobsWrapper == nil {
					if debug >= 4 {
						log.Printf("Serial port closed; points not sent: %v", pointsToSend)
					}
					return
				}
				sd.wrSeq++
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

	_, err = sd.portCobsWrapper.Write(d)
	if err != nil {
		return fmt.Errorf("error writing data to port: %w", err)
	}

	sd.config.Tx++
	err = SendPoints(sd.nc, sd.natsSub,
		data.Points{{Type: data.PointTypeTx, Value: float64(sd.config.Tx)}},
		false)

	if err != nil {
		return fmt.Errorf("Error sending Serial tx stats: %w", err)
	}

	if !ack {
		_ = ack
		// TODO: we need to check for response and implement retries
		// yet.
	}

	return nil
}

type downloadState struct {
	buf          bytes.Buffer
	seq          byte
	currentBlock int
}

// Run the main logic for this client and blocks until stopped
func (sd *SerialDevClient) Run() error {
	log.Println("Starting serial client:", sd.config.Description)

	if sd.config.Connected {
		sd.config.Connected = false
		err := SendNodePoint(sd.nc, sd.config.ID, data.Point{Type: data.PointTypeConnected, Value: 0}, false)
		if err != nil {
			log.Println("Error sending connected point")
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("Error creating a new fsnotify watcher: %w", err)

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

	serialReadData := make(chan []byte)
	listenerClosed := make(chan struct{})
	listenerSerialErr := make(chan struct{})

	closePort := func() {
		if sd.portCobsWrapper != nil {
			log.Println("Closing serial port:", sd.config.Description)
			sd.portCobsWrapper.Close()
		}
		sd.portCobsWrapper = nil

		sd.config.Connected = false
		err := SendNodePoint(sd.nc, sd.config.ID, data.Point{Type: data.PointTypeConnected, Value: 0}, false)
		if err != nil {
			log.Println("Error sending connected point")
		}
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
				data.Points{{Type: data.PointTypeMaxMessageLength, Value: 1024}}, true)
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

			serialPort, err := serial.Open(sd.config.Port, mode)
			if err != nil {
				log.Printf("Error opening serial port %v: %v", sd.config.Description,
					err)
				timerCheckPort.Reset(checkPortDur)
				return
			}

			time.Sleep(time.Millisecond)
			err = serialPort.SetDTR(false)
			if err != nil {
				log.Printf("Error clearing serial port DTR: %v\n", err)
			}

			time.Sleep(time.Millisecond * 100)
			err = serialPort.SetDTR(true)
			if err != nil {
				log.Printf("Error setting serial port DTR: %v\n", err)
			}

			io = serialPort
		}

		sd.portCobsWrapper = NewCobsWrapper(io, sd.config.MaxMessageLength)
		sd.portCobsWrapper.SetDebug(sd.config.Debug)
		timerCheckPort.Stop()

		log.Println("Serial port opened:", sd.config.Description)

		go listener(sd.portCobsWrapper, sd.config.MaxMessageLength)

		p := data.Points{{
			Time: time.Now(),
			Type: data.PointTypeTimeSync,
		}}

		sd.config.Connected = true
		err := SendNodePoint(sd.nc, sd.config.ID, data.Point{Type: data.PointTypeConnected, Value: 1}, false)
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
		case <-listenerClosed:
			closePort()
			timerCheckPort.Reset(checkPortDur)
		case <-listenerSerialErr:
			sd.config.ErrorCount++
			points := []data.Point{{Type: data.PointTypeErrorCount, Value: float64(sd.config.ErrorCount)}}
			err := SendPoints(sd.nc, sd.natsSub, points, false)
			if err != nil {
				log.Println("Error sending error points:", err)
			}
		case e, ok := <-watcher.Events:
			if ok {
				if e.Name == sd.config.Port {
					if e.Op == fsnotify.Remove {
						closePort()
					} else if e.Op == fsnotify.Create {
						openPort()
					}
				}
			}
		case rd := <-serialReadData:
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

				err := SendPoints(sd.nc, sd.natsSub, []data.Point{{Type: t, Value: float64(cnt)}}, false)
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
				points := data.Points{{Type: data.PointTypeLog, Text: string(payload)}}

				if sd.config.Debug >= 1 {
					log.Printf("Serial client %v: log: %v\n",
						sd.config.Description, string(payload))
				}
				err := SendPoints(sd.nc, sd.natsSubSerialPoints, points, false)
				if err != nil {
					log.Println("Error sending log point from MCU:", err)
				}
			}

			// we must have a protobuf payload
			points, errDecode := data.PbDecodeSerialPoints(payload)
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
					data.Point{Type: data.PointTypeErrorCount, Value: float64(sd.config.ErrorCount)})
			}

			if time.Since(sd.lastSendStats) > time.Second*5 {
				adminPoints = append(adminPoints,
					data.Points{
						{Time: time.Now(), Type: data.PointTypeRx, Value: float64(sd.config.Rx)},
						{Time: time.Now(), Type: data.PointTypeHrRx, Value: float64(sd.config.HrRx)},
					}...)
				sd.lastSendStats = time.Now()
			}

			if time.Since(sd.rateLastSend) > time.Second {
				now := time.Now()
				elapsedSec := now.Sub(sd.rateLastSend).Seconds()
				rate := float64(sd.ratePointCount) / elapsedSec
				rateHR := float64(sd.ratePointCountHR) / elapsedSec
				adminPoints = append(adminPoints,
					data.Point{Time: now, Type: data.PointTypeRate,
						Value: rate},
					data.Point{Time: now, Type: data.PointTypeRateHR,
						Value: rateHR},
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

		case pts := <-sd.newPoints:
			op := false
			updateNatsSubjects := false
			for _, p := range pts.Points {
				// check if any of the config changes should cause us to re-open the port
				if p.Type == data.PointTypePort ||
					p.Type == data.PointTypeBaud ||
					p.Type == data.PointTypeDisabled ||
					p.Type == data.PointTypeMaxMessageLength {
					op = true
				}

				if p.Type == data.PointTypePort {
					err := watcher.Add(filepath.Dir(p.Text))
					if err != nil {
						log.Println("Error adding watcher on serial port name change:", p.Text)
					}
				}

				if p.Type == data.PointTypeDisabled {
					if p.Value == 0 {
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

				if p.Type == data.PointTypeDebug {
					sd.portCobsWrapper.SetDebug(int(p.Value))
				}

				if p.Type == data.PointTypeDownload {
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

			if sd.portCobsWrapper == nil {
				break
			}

			if sd.config.ErrorCountReset {
				points := data.Points{
					{Type: data.PointTypeErrorCount, Value: 0},
					{Type: data.PointTypeErrorCountReset, Value: 0},
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
					{Type: data.PointTypeErrorCountHR, Value: 0},
					{Type: data.PointTypeErrorCountResetHR, Value: 0},
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
					{Type: data.PointTypeRx, Value: 0},
					{Type: data.PointTypeRxReset, Value: 0},
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
					{Type: data.PointTypeHrRx, Value: 0},
					{Type: data.PointTypeHrRxReset, Value: 0},
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
					{Type: data.PointTypeTx, Value: 0},
					{Type: data.PointTypeTxReset, Value: 0},
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
						data.PointTypeTxReset:
						continue
					}

					// strip off Origin as MCU does not need that
					p.Origin = ""
					toSend = append(toSend, p)
				}

				if len(toSend) > 0 {
					sd.wrSeq++
					err := sd.sendPointsToDevice(sd.wrSeq, false, "", toSend)
					if err != nil {
						log.Println("Error sending points to serial device:", err)
					}
				}
			}

		case pts := <-sd.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &sd.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}

		case sData := <-sd.sendPointsCh:
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
