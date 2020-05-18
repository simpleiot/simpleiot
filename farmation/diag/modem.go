package diag

import (
	"errors"
	"time"

	"github.com/cbrake/go-serial/serial"
	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/file"
	"github.com/simpleiot/simpleiot/network"
	"github.com/simpleiot/simpleiot/respreader"
)

type modemUsb struct{}

func (d modemUsb) String() string {
	return "modem-usb"
}

const modemUsbDev = "/dev/ttyUSB0"

func (d modemUsb) Run() error {
	if !file.Exists(modemUsbDev) {
		return errors.New("modem not detected on USB")
	}

	isio.ResetModem()

	time.Sleep(10 * time.Millisecond)
	if file.Exists(modemUsbDev) {
		return errors.New("modem still detected after reset")
	}

	time.Sleep(7 * time.Second)

	if !file.Exists(modemUsbDev) {
		return errors.New("Modem is not detected on USB bus after reset")
	}

	return nil
}

type modemSerial struct{}

func (d modemSerial) String() string {
	return "modem-serial"
}

func (d modemSerial) Run() error {
	options := serial.OpenOptions{
		PortName:          isio.SerialModem,
		BaudRate:          115200,
		DataBits:          8,
		StopBits:          1,
		MinimumReadSize:   1,
		RTSCTSFlowControl: true,
	}

	port, err := serial.Open(options)

	if err != nil {
		return err
	}

	port = respreader.NewReadWriteCloser(port, 1*time.Second,
		50*time.Millisecond)

	err = network.CmdOK(port, "AT")

	if err != nil {
		return err
	}

	return nil
}

func init() {
	Register(modemUsb{})
	Register(modemSerial{})
}
