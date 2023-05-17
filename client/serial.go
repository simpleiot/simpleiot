package client

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/file"
	"github.com/simpleiot/simpleiot/test"
	"go.bug.st/serial"
)

// SerialDev represents a serial (MCU) config
type SerialDev struct {
	ID               string `node:"id"`
	Parent           string `node:"parent"`
	Description      string `point:"description"`
	Port             string `point:"port"`
	Baud             string `point:"baud"`
	MaxMessageLength int    `point:"maxMessageLength"`
	Debug            int    `point:"debug"`
	Disable          bool   `point:"disable"`
	Log              string `point:"log"`
	Rx               int    `point:"rx"`
	RxReset          bool   `point:"rxReset"`
	Tx               int    `point:"tx"`
	TxReset          bool   `point:"txReset"`
	Uptime           int    `point:"uptime"`
	ErrorCount       int    `point:"errorCount"`
	ErrorCountReset  bool   `point:"errorCountReset"`
	Rate             bool   `point:"rate"`
	Connected        bool   `point:"connected"`
}

// SerialDevClient is a SIOT client used to manage serial devices
type SerialDevClient struct {
	nc             *nats.Conn
	config         SerialDev
	stop           chan struct{}
	newPoints      chan NewPoints
	newEdgePoints  chan NewPoints
	wrSeq          byte
	lastSendStats  time.Time
	natsSub        string
	natsSubHR      string
	natsSubHRUp    string
	ratePointCount int
	rateLastSend   time.Time
}

// NewSerialDevClient ...
func NewSerialDevClient(nc *nats.Conn, config SerialDev) Client {
	return &SerialDevClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		natsSub:       SubjectNodePoints(config.ID),
		natsSubHR:     fmt.Sprintf("phr.%v", config.ID),
		natsSubHRUp:   fmt.Sprintf("phrup.%v.%v", config.Parent, config.ID),
	}
}

// Run the main logic for this client and blocks until stopped
func (sd *SerialDevClient) Run() error {
	log.Println("Starting serial client: ", sd.config.Description)

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
			log.Println("Error adding watcher for: ", sd.config.Port)
		}
	}

	checkPortDur := time.Second * 10
	timerCheckPort := time.NewTimer(checkPortDur)

	var port *CobsWrapper
	serialReadData := make(chan []byte)
	listenerClosed := make(chan struct{})

	closePort := func() {
		if port != nil {
			log.Println("Closing serial port: ", sd.config.Description)
			port.Close()
		}
		port = nil

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

					// we don't want to reset the port on every COBS
					// decode error, so accumulate a few before we do this
					if err == ErrCobsDecodeError ||
						err == ErrCobsTooMuchData {
						errCount++
						if errCount < 50 {
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

			buf = buf[0:c]
			serialReadData <- buf
		}
	}

	sendPointsToDevice := func(sub string, pts data.Points) error {
		sd.wrSeq++
		d, err := SerialEncode(sd.wrSeq, sub, pts)
		if err != nil {
			return fmt.Errorf("error encoding points to send to MCU: %w", err)
		}

		if sd.config.Debug >= 4 {
			log.Printf("SER TX (%v) seq:%v :\n%v", sd.config.Description, sd.wrSeq, sub)
		}

		_, err = port.Write(d)
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

		// TODO: we need to check for response and implement retries
		// yet.

		return nil
	}

	openPort := func() {
		if sd.config.MaxMessageLength <= 0 {
			sd.config.MaxMessageLength = 1024
			err := SendPoints(sd.nc, sd.natsSub,
				data.Points{{Type: data.PointTypeMaxMessageLength, Value: 1024}}, true)
			if err != nil {
				log.Println("Error sending max message len message: ", err)
			}
		}

		// make sure port is closed before we try to (re)open it
		closePort()

		if sd.config.Disable {
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
				log.Println("SerialDevClient: error opening fifo: ", err)
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

			io = serialPort
		}

		port = NewCobsWrapper(io, sd.config.MaxMessageLength)
		port.SetDebug(sd.config.Debug)
		timerCheckPort.Stop()

		log.Println("Serial port opened: ", sd.config.Description)

		go listener(port, sd.config.MaxMessageLength)

		p := data.Points{{
			Time: time.Now(),
			Type: data.PointTypeTimeSync,
		}}

		sd.config.Connected = true
		err := SendNodePoint(sd.nc, sd.config.ID, data.Point{Type: data.PointTypeConnected, Value: 1}, false)
		if err != nil {
			log.Println("Error sending connected point")
		}

		err = sendPointsToDevice("", p)
		if err != nil {
			log.Println("Error sending time sync point to device: %w", err)
		}
	}

	if file.Exists(sd.config.Port) {
		openPort()
	}

	for {
		select {
		case <-sd.stop:
			log.Println("Stopping serial client: ", sd.config.Description)
			closePort()
			return nil
		case <-timerCheckPort.C:
			openPort()
		case <-listenerClosed:
			closePort()
			timerCheckPort.Reset(checkPortDur)
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
				log.Println("SER RX: ", test.HexDump(rd))
			}

			// decode serial packet
			seq, subject, payload, err := SerialDecode(rd)
			if err != nil {
				log.Println("Serial framing error: ", err)
				break
			}

			if subject == "phr" {
				// we have high rate points
				err := sd.nc.Publish(sd.natsSubHRUp, payload)
				if err != nil {
					log.Println("Error publishing HR data: ", err)
				}
				// we're done
				break
			}

			if subject == "log" {
				points := data.Points{{Type: data.PointTypeLog, Text: string(payload)}}

				if sd.config.Debug >= 1 {
					log.Printf("Serial client %v: log: %v\n",
						sd.config.Description, string(payload))
				}
				err := SendPoints(sd.nc, sd.natsSub, points, false)
				if err != nil {
					log.Println("Error sending log point from MCU: ", err)
				}
			}

			// we must have a protobuf payload
			points, errDecode := data.PbDecodeSerialPoints(payload)

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
					log.Printf("SER RX (%v) seq:%v\n%v", sd.config.Description, seq, points)
				}

				// send response
				err := sendPointsToDevice("ack", nil)
				if err != nil {
					log.Println("Error sending ack to device: ", err)
				}

				err = data.MergePoints(sd.config.ID, points, &sd.config)
				if err != nil {
					log.Println("error merging new points: ", err)
				}
			} else {
				log.Println("Error decoding serial packet from device: ",
					sd.config.Description, errDecode)
				sd.config.ErrorCount++
				points = append(points,
					data.Point{Type: data.PointTypeErrorCount, Value: float64(sd.config.ErrorCount)})
			}

			if time.Since(sd.lastSendStats) > time.Second*5 {
				points = append(points,
					data.Point{Time: time.Now(), Type: data.PointTypeRx, Value: float64(sd.config.Rx)})
				sd.lastSendStats = time.Now()
			}

			if time.Since(sd.rateLastSend) > time.Second {
				now := time.Now()
				rate := float64(sd.ratePointCount) / now.Sub(sd.rateLastSend).Seconds()
				points = append(points,
					data.Point{Time: now, Type: data.PointTypeRate,
						Value: rate})
				sd.rateLastSend = now
				sd.ratePointCount = 0
			}

			if len(points) > 0 {
				err := SendPoints(sd.nc, sd.natsSub, points, false)
				if err != nil {
					log.Println("Error sending points received from MCU: ", err)
				}
			}

		case pts := <-sd.newPoints:
			op := false
			for _, p := range pts.Points {
				// check if any of the config changes should cause us to re-open the port
				if p.Type == data.PointTypePort ||
					p.Type == data.PointTypeBaud ||
					p.Type == data.PointTypeDisable ||
					p.Type == data.PointTypeMaxMessageLength {
					op = true
				}

				if p.Type == data.PointTypePort {
					err := watcher.Add(filepath.Dir(p.Text))
					if err != nil {
						log.Println("Error adding watcher on serial port name change: ", p.Text)
					}
				}

				if p.Type == data.PointTypeDisable {
					if p.Value == 0 {
						closePort()
					} else {
						op = true
					}
				}
			}

			err := data.MergePoints(pts.ID, pts.Points, &sd.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
			if op {
				openPort()
			}

			if port == nil {
				break
			}

			if sd.config.ErrorCountReset {
				points := data.Points{
					{Type: data.PointTypeErrorCount, Value: 0},
					{Type: data.PointTypeErrorCountReset, Value: 0},
				}
				err = SendPoints(sd.nc, sd.natsSub, points, false)
				if err != nil {
					log.Println("Error resetting MCU error count: ", err)
				}

				sd.config.ErrorCountReset = false
				sd.config.ErrorCount = 0
			}

			if sd.config.RxReset {
				points := data.Points{
					{Type: data.PointTypeRx, Value: 0},
					{Type: data.PointTypeRxReset, Value: 0},
				}
				err = SendPoints(sd.nc, sd.natsSub, points, false)
				if err != nil {
					log.Println("Error resetting MCU error count: ", err)
				}

				sd.config.RxReset = false
				sd.config.Rx = 0
			}

			if sd.config.TxReset {
				points := data.Points{
					{Type: data.PointTypeTx, Value: 0},
					{Type: data.PointTypeTxReset, Value: 0},
				}
				err = SendPoints(sd.nc, sd.natsSub, points, false)
				if err != nil {
					log.Println("Error resetting MCU error count: ", err)
				}

				sd.config.TxReset = false
				sd.config.Tx = 0
			}

			// check if we have any points that need sent to MCU
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
				case data.PointTypeDebug:
					port.SetDebug(int(p.Value))
				}

				// strip off Origin as MCU does not need that
				p.Origin = ""
				toSend = append(toSend, p)
			}

			if len(toSend) > 0 {
				err := sendPointsToDevice("", toSend)
				if err != nil {
					log.Println("Error sending points to serial device: ", err)
				}
			}

		case pts := <-sd.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &sd.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

			// TODO need to send edge points to MCU, not implemented yet
		}
	}
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
