package diag

import (
	"errors"
	"fmt"
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
	time.Sleep(30 * time.Millisecond)
	isio.GpioOut(isio.GpioRadioReset, false)
	time.Sleep(200 * time.Millisecond)

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

	commandMode := "+++"

	n, err := p.Write([]byte(commandMode))
	if err != nil {
		return err
	}

	fmt.Println("wrote: ", n)

	if n != len(commandMode) {
		return errors.New("write count is wrong")
	}

	time.Sleep(1200 * time.Millisecond)

	readString := make([]byte, 100)
	n, err = p.Read(readString)

	if err != nil {
		return err
	}

	fmt.Println("read: ", n)

	readString = readString[:n]

	if string(readString) == "OK" {
		return errors.New("Expected OK, got: " + string(readString))
	}

	fmt.Println("readString: ", string(readString))

	return nil
}

func init() {
	Register(radio{})
}
