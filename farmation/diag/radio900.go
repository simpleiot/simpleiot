package diag

import (
	"time"

	"github.com/cbrake/go-serial/serial"
	"github.com/simpleiot/simpleiot/farmation/isio"
)

type radio struct{}

func (d radio) String() string {
	return "radio"
}

func (d radio) Run() error {
	isio.GpioOut(isio.GpioRadioSleep, false)
	isio.GpioOut(isio.GpioRadioReset, true)
	time.Sleep(100 * time.Millisecond)

	options := serial.OpenOptions{
		PortName:              isio.SerialRadio,
		BaudRate:              9600,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       1,
		InterCharacterTimeout: 200,
		RTSCTSFlowControl:     false,
	}

	p, err := serial.Open(options)
	if err != nil {
		return err
	}

	defer p.Close()

	isio.GpioOut(isio.GpioRadioReset, false)
	time.Sleep(200 * time.Millisecond)

	return DigiCheckAt(p)
}

func init() {
	Register(radio{})
}
