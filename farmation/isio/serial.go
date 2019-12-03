package isio

import (
	"io"
	"time"

	"github.com/cbrake/go-serial/serial"
	"github.com/simpleiot/simpleiot/respreader"
)

// defines for serial ports on IS
const (
	SerialConsole    string = "/dev/ttyS0"
	SerialGps        string = "/dev/ttyS1"
	SerialDebug      string = "/dev/ttyS2"
	SerialRS232RS485 string = "/dev/ttyS3"
	SerialModem      string = "/dev/ttyS4"
	SerialRadio      string = "/dev/ttyS5"
)

// OpenSerialModem opens the modem serial port
func OpenSerialModem() (io.ReadWriteCloser, error) {
	ResetModem()

	options := serial.OpenOptions{
		PortName:              SerialModem,
		BaudRate:              9600,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       1,
		InterCharacterTimeout: 0,
		RTSCTSFlowControl:     true,
	}

	port, err := serial.Open(options)

	if err != nil {
		return nil, err
	}

	return respreader.NewResponseReadWriteCloser(port, 2*time.Second,
		50*time.Millisecond), nil
}
