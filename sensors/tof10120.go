package sensors

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// The TOF10120 is a Laser distance sensing (TOF) module
// Chinese datasheet and Google translate versions of the datasheet
// can be found: https://github.com/simpleiot/reference/tree/master/sensors

// Wiring of TOF10120 to FTDI cable:
// TOF10120	  FTDI
// 1 (black, GND) 1 (blk, GND)
// 2 (red, VDD)	  3 (red, VCC)
// 3 (yel, RXD)	  4 (org, TXD)
// 4 (wht, TXD)	  5 (yel, RXD)

// TOF10120 is a driver for a TOF10120 sensor
type TOF10120 struct {
	port io.ReadWriter
}

// NewTOF10120 creates a instance to initialize and read the TOF sensor
// port must return an entire packet for each Read().
// github.com/simpleiot/simpleiot/respreader is a good
// way to do this.
func NewTOF10120(port io.ReadWriter) *TOF10120 {
	return &TOF10120{
		port: port,
	}
}

var re = regexp.MustCompile(`^([0-9]*)mm`)

// SetSendInterval sets the interval at which sensor sends data
// 10-9999ms, default 100ms
// this should be called before Read() is started
func (tof *TOF10120) SetSendInterval(interval int) error {
	// try to fit this between two send intervals
	buf := make([]byte, 100)
	c, err := tof.port.Read(buf)

	if err != nil {
		return err
	}

	if c <= 0 {
		return errors.New("Sensor does not seem to be active")
	}

	wr := fmt.Sprintf("s2-%v#", interval)

	_, err = tof.port.Write([]byte(wr))

	if err != nil {
		return err
	}

	c, err = tof.port.Read(buf)
	if err != nil {
		return err
	}

	if c <= 0 {
		return errors.New("Sensor did not return any data after write")
	}

	buf = buf[:c]

	if !strings.Contains(string(buf), "ok") {
		return errors.New("Sensor did not return ok")
	}

	return nil
}

// Read returns the distance in mm. The sensor continuously ouputs
// readings so the callback is called each time a new reading is
// read.
func (tof *TOF10120) Read(dataCallback func(dist int),
	errCallback func(err error)) error {
	for {
		if tof.port == nil {
			return errors.New("no port")
		}

		buf := make([]byte, 100)
		c, err := tof.port.Read(buf)
		if err != nil {
			if err != io.EOF {
				return err
			}
		}

		if c <= 0 {
			continue
		}

		buf = buf[:c]

		matches := re.FindSubmatch(buf)

		if len(matches) < 2 {
			errCallback(errors.New("error parsing data"))
			continue
		}

		v, err := strconv.Atoi(string(matches[1]))

		if err != nil {
			errCallback(err)
			continue
		}

		dataCallback(v)
	}
}
