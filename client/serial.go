package client

import (
	"io"
	"log"
	"strconv"
	"time"
	"unicode"

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
	Debug       int    `point:"debug"`
	Disable     bool   `point:"disable"`
	Log         string `point:"log"`
	Rx          int    `point:"rx"`
	Tx          int    `point:"tx"`
	Uptime      int    `point:"uptime"`
	ErrorCount  int    `point:"errorCount"`
}

// SerialDevClient is a SIOT client used to manage serial devices
type SerialDevClient struct {
	nc            *nats.Conn
	config        SerialDev
	stop          chan struct{}
	newPoints     chan []data.Point
	newEdgePoints chan []data.Point
	wrSeq         byte
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
	serialReadData := make(chan []byte)
	listenerClosed := make(chan struct{})

	closePort := func() {
		if port != nil {
			log.Println("Closing port ", sd.config.Description)
			port.Close()
		}
		port = nil
	}

	listener := func(port io.ReadWriteCloser) {
		errCount := 0
		for {
			buf := make([]byte, 1024)
			c, err := port.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading port %v: %v\n", sd.config.Description, err)

					// we don't want to reset the port on every COBS
					// decode error, so accumulate a few before we do this
					if err == ErrCobsDecodeError {
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

		port = NewCobsWrapper(io)
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
		case rd := <-serialReadData:
			sd.config.Rx++
			rxPt := data.Point{Type: data.PointTypeRx, Value: float64(sd.config.Rx)}
			// figure out if the data is ascii string or points
			// try pb decode
			seq, subject, points, err := SerialDecode(rd)
			if err == nil && len(points) > 0 {
				if subject == "" {
					subject = SubjectNodePoints(sd.config.ID)
				}
				points = append(points, rxPt)
				// send response
				d, err := SerialEncode(seq, "", nil)
				if err != nil {
					log.Println("Error enoding serial response: ", err)
				} else {
					_, err := port.Write(d)
					if err != nil {
						log.Println("Error writing response to port: ", err)
					}
				}
				err = data.MergePoints(points, &sd.config)
				if err != nil {
					log.Println("error merging new points: ", err)
				}

			} else {
				subject = SubjectNodePoints(sd.config.ID)
				// check if ascii
				isASCII := true
				for i := 0; i < len(rd); i++ {
					if rd[i] > unicode.MaxASCII {
						isASCII = false
						break
					}
				}
				if isASCII {
					points = data.Points{
						rxPt,
						{Type: data.PointTypeLog, Text: string(rd)},
					}
					log.Printf("Serial client %v: log: %v\n", sd.config.Description, string(rd))
				} else {
					log.Println("Error decoding serial packet from device: ",
						sd.config.Description)
					sd.config.ErrorCount++
					points = data.Points{
						rxPt,
						{Type: data.PointTypeErrorCount, Value: float64(sd.config.ErrorCount)},
					}
				}
			}

			err = SendPoints(sd.nc, subject, points, false)
			if err != nil {
				log.Println("Error sending points received from MCU: ", err)
			}
		case pts := <-sd.newPoints:
			// we only process points whose origin is set, IE did not originate
			// from the serial device and are simply echo'd back.
			toMerge := data.Points{}
			op := false
			for _, p := range pts {
				if p.Origin != "" {
					toMerge = append(toMerge, p)
					if p.Type == data.PointTypePort ||
						p.Type == data.PointTypeBaud ||
						p.Type == data.PointTypeDisable {
						op = true
						break
					}
				}
			}

			err := data.MergePoints(toMerge, &sd.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
			if op {
				openPort()
			}

			// check if we have any points that need sent to MCU
			toSend := data.Points{}
			for _, p := range toMerge {
				switch p.Type {
				case data.PointTypePort,
					data.PointTypeBaud,
					data.PointTypeDescription:
					continue
				}

				// strip off Origin as MCU does not need that
				p.Origin = ""
				toSend = append(toSend, p)
			}

			if len(toSend) > 0 {
				sd.wrSeq++
				d, err := SerialEncode(sd.wrSeq, "", toSend)
				if err != nil {
					log.Println("error encoding points to send to MCU: ", err)
				}

				_, err = port.Write(d)
				if err != nil {
					log.Println("error writing data to port: ", err)
				}

				// TODO: we need to check for response and implement retries
				// yet.
			}

		case pts := <-sd.newEdgePoints:
			// we only process points whose origin is set, IE did not originate
			// from the serial device and are simply echo'd back.
			toMerge := data.Points{}
			for _, p := range pts {
				if p.Origin != "" {
					toMerge = append(toMerge, p)
				}
			}

			err := data.MergeEdgePoints(toMerge, &sd.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

			// TODO need to send edge points to MCU, not implemented yet
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
