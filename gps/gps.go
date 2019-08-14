package gps

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	nmea "github.com/adrianmo/go-nmea"
	"github.com/cbrake/go-serial/serial"
	"github.com/simpleiot/simpleiot/data"
)

// Gps represent a GPS receiver
type Gps struct {
	portName string
	baud     uint
	c        chan data.GpsPos
	debug    bool
	stop     bool
	port     io.ReadWriteCloser
}

// NewGps is used to create a new Gps type
func NewGps(portName string, baud uint, c chan data.GpsPos) *Gps {
	return &Gps{
		portName: portName,
		baud:     baud,
		c:        c,
	}
}

// SetDebug can be used to turn debugging on and off
func (gps *Gps) SetDebug(d bool) {
	gps.debug = d
}

// Stop can be used to stop the GPS aquisition and close port
func (gps *Gps) Stop() {
	gps.port.Close()
	gps.stop = true
}

// Start is used to start reading the GPS, and data will be sent back
// through the handler
func (gps *Gps) Start() {
	gps.stop = false
	go func() {
		options := serial.OpenOptions{
			PortName:              gps.portName,
			BaudRate:              gps.baud,
			DataBits:              8,
			StopBits:              1,
			MinimumReadSize:       1,
			InterCharacterTimeout: 0,
		}
		for {
			var err error
			gps.port, err = serial.Open(options)

			if err != nil {
				if gps.debug {
					fmt.Println("failed to open port: ", options.PortName)
				}
				// delay a bit before trying to open port again
				time.Sleep(10 * time.Second)
				continue
			}

			fmt.Println("GPS port opened: ", options.PortName)
			reader := bufio.NewReader(gps.port)

			for {
				if gps.stop {
					fmt.Println("Closing GPS")
					return
				}

				line, err := reader.ReadString('\n')

				if gps.debug {
					fmt.Println(line)
				}

				if err != nil {
					fmt.Println("Error reading gps, closing: ", err)
					break
				}

				s, err := nmea.Parse(strings.TrimSpace(line))
				if err == nil {
					if s.DataType() == nmea.TypeGGA {
						m := s.(nmea.GGA)
						ret := data.GpsPos{
							Lat:    m.Latitude,
							Long:   m.Longitude,
							Fix:    m.FixQuality,
							NumSat: m.NumSatellites,
						}

						gps.c <- ret
					}
				} else {
					if gps.debug {
						fmt.Println("Error parsing GPS data: ", err)
					}
				}
			}

			gps.port.Close()
			if gps.stop {
				fmt.Println("Closing GPS")
				return
			}

			// delay a bit before trying to open port again
			time.Sleep(10 * time.Second)
		}
	}()
}
