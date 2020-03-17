package diag

import (
	"errors"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/file"
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
	p, err := isio.OpenSerialModem()
	if err != nil {
		return err
	}

	// give modem a few seconds to power up
	time.Sleep(6 * time.Second)

	defer p.Close()

	return DigiCheckAt(p)
}

func init() {
	Register(modemUsb{})
	Register(modemSerial{})
}
