package diag

import (
	"errors"
	"fmt"
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
	isio.GpioOut(isio.GpioModemReset, false)

	// modem takes about 5 seconds to show up on USB bus after reset
	time.Sleep(5 * time.Second)

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

	commandMode := "+++"

	n, err := p.Write([]byte(commandMode))
	if err != nil {
		return err
	}

	fmt.Println("wrote: ", n)

	if n != len(commandMode) {
		return errors.New("write count is wrong")
	}

	time.Sleep(1500 * time.Millisecond)

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

	fmt.Println("flooding port with data")

	writeBuf := make([]byte, 1024)
	n, err = p.Write(writeBuf)

	fmt.Println("wrote: ", n)

	return nil
}

func init() {
	Register(modem{})
}
