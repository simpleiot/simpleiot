package diag

import (
	"errors"
	"time"

	"github.com/cbrake/go-serial/serial"
	"github.com/simpleiot/simpleiot/farmation/isio"
)

type modem struct{}

func (d modem) String() string {
	return "modem"
}

func (d modem) Run() error {
	isio.GpioOut(isio.GpioModemSleep, false)
	isio.GpioOut(isio.GpioModemReset, true)
	time.Sleep(30 * time.Millisecond)

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

	err = DigiCheckAt(p)

	if err == nil {
		return errors.New("Modem is reponding when reset is asserted")
	}

	isio.GpioOut(isio.GpioModemReset, false)

	// modem takes about 5 seconds to show up on USB bus after reset
	time.Sleep(5 * time.Second)

	DigiCheckAt(p)
	return DigiCheckAt(p)
}

func init() {
	Register(modem{})
}
