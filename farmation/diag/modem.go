package diag

import (
	"errors"
	"time"

	"github.com/cbrake/go-serial/serial"
	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/file"
)

type modemUsb struct{}

func (d modemUsb) String() string {
	return "modemUsb"
}

const modemUsbDev = "/dev/cdc-wdm0"

func (d modemUsb) Run() error {
	isio.GpioOut(isio.GpioModemSleep, false)
	time.Sleep(2 * time.Second)

	if file.Exists(modemUsbDev) {
		return errors.New("modem reset is not working")
	}

	options := serial.OpenOptions{
		PortName:              isio.SerialModem,
		BaudRate:              9600,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       1,
		InterCharacterTimeout: 200,
		RTSCTSFlowControl:     true,
	}

	p, err := serial.Open(options)
	if err != nil {
		return err
	}

	defer p.Close()

	// modem takes about 5 seconds to show up on USB bus after reset
	time.Sleep(6 * time.Second)

	if !file.Exists(modemUsbDev) {
		return errors.New("Modem is not detected on USB bus")
	}

	return DigiCheckAt(p)
}

type modemSerial struct{}

func (d modemSerial) String() string {
	return "modemSerial"
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
