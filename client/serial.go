package client

import (
	"io"
	"log"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/respreader"
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
	Debug       int    `point:"debug"`
	Disable     bool   `point:"disable"`
	Rx          int    `point:"rx"`
	Tx          int    `point:"tx"`
}

// SerialDevClient is a SIOT client used to manage serial devices
type SerialDevClient struct {
	nc            *nats.Conn
	config        SerialDev
	stop          chan struct{}
	newPoints     chan []data.Point
	newEdgePoints chan []data.Point
}

// NewSerialDevClient ...
func NewSerialDevClient(nc *nats.Conn, config SerialDev) Client {
	return &SerialDevClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan []data.Point),
		newEdgePoints: make(chan []data.Point),
	}
}

// Start runs the main logic for this client and blocks until stopped
func (sd *SerialDevClient) Start() error {
	log.Println("Starting serial client: ", sd.config.Description)

	checkPortDur := time.Second * 10
	timerCheckPort := time.NewTimer(checkPortDur)

	var port io.ReadWriteCloser
	readData := make(chan []byte)
	listenerClosed := make(chan struct{})

	closePort := func() {
		if port != nil {
			port.Close()
		}
		port = nil
	}

	listener := func(port io.ReadWriteCloser) {
		for {
			buf := make([]byte, 1024)
			c, err := port.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading port %v: %v\n", sd.config.Description, err)
					listenerClosed <- struct{}{}
					return
				}
			}
			if c <= 0 {
				continue
			}

			buf = buf[0:c]
			readData <- buf
		}
	}

	openPort := func() {
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

		port = respreader.NewReadWriteCloser(io, time.Millisecond*100, time.Millisecond*20)
		timerCheckPort.Stop()

		go listener(port)
	}

	openPort()

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
		case rd := <-readData:
			sd.config.Rx++
			err := SendNodePoint(sd.nc, sd.config.ID,
				data.Point{Type: data.PointTypeRx, Value: float64(sd.config.Rx)}, false)
			if err != nil {
				log.Println("Error sending rx stats: ", err)
			}
			log.Printf("Serial client %v: log: %v\n", sd.config.Description, string(rd))
		case pts := <-sd.newPoints:
			err := data.MergePoints(pts, &sd.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
			for _, p := range pts {
				if p.Type == data.PointTypePort ||
					p.Type == data.PointTypeBaud ||
					p.Type == data.PointTypeDisable {
					openPort()
					break
				}
			}
		case pts := <-sd.newEdgePoints:
			err := data.MergeEdgePoints(pts, &sd.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
		}
	}
}

// Stop sends a signal to the Start function to exit
func (sd *SerialDevClient) Stop(err error) {
	close(sd.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (sd *SerialDevClient) Points(points []data.Point) {
	sd.newPoints <- points
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (sd *SerialDevClient) EdgePoints(points []data.Point) {
	sd.newEdgePoints <- points
}