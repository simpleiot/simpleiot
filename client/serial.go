package client

import (
	"io"
	"log"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/respreader"
	"go.bug.st/serial"
)

type serialDev struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Port        string `point:"port"`
	Baud        string `point:"baud"`
	Debug       int    `point:"debug"`
	Disable     bool   `point:"disable"`
}

type serialDevClient struct {
	nc            *nats.Conn
	config        serialDev
	stop          chan struct{}
	newPoints     chan []data.Point
	newEdgePoints chan []data.Point
}

func newSerialDevClient(nc *nats.Conn, config serialDev) Client {
	return &serialDevClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan []data.Point),
		newEdgePoints: make(chan []data.Point),
	}
}

// Start runs the main logic for this client and blocks until stopped
func (sd *serialDevClient) Start() error {
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

		port = respreader.NewReadWriteCloser(serialPort, time.Millisecond*100, time.Millisecond*20)
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
			log.Printf("Serial client %v debug: %v\n", sd.config.Description, string(rd))
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
func (sd *serialDevClient) Stop(err error) {
	close(sd.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (sd *serialDevClient) Points(points []data.Point) {
	sd.newPoints <- points
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (sd *serialDevClient) EdgePoints(points []data.Point) {
	sd.newEdgePoints <- points
}
