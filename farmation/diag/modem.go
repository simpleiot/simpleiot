package diag

import (
	"errors"
	"time"

	"github.com/cbrake/go-serial/serial"
	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/file"
)

type modem struct{}

func (d modem) String() string {
	return "modem"
}

const modemUsbDev = "/dev/cdc-wdm0"

func (d modem) Run() error {
	isio.GpioOut(isio.GpioModemSleep, false)
	isio.GpioOut(isio.GpioModemReset, true)
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

	isio.GpioOut(isio.GpioModemReset, false)

	// modem takes about 5 seconds to show up on USB bus after reset
	time.Sleep(6 * time.Second)

	if !file.Exists(modemUsbDev) {
		return errors.New("Modem is not detected on USB bus")
	}

	return DigiCheckAt(p)
}

func init() {
	Register(modem{})
}
