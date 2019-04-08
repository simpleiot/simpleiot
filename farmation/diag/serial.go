package diag

import (
	"errors"
	"fmt"
	"time"

	"github.com/cbrake/go-serial/serial"
	"github.com/simpleiot/simpleiot/farmation/isio"
)

type rs232 struct{}

func (d rs232) String() string {
	return "rs232"
}

func (d rs232) Run() error {
	GetInput("rs232 loopback: connect Main pin #7 to #8")
	isio.GpioOut(isio.GpioSerialShutdown, false)
	isio.GpioOut(isio.GpioSerialLoopback, false)
	isio.GpioOut(isio.GpioSerialRsSelectRs485, false)

	options := serial.OpenOptions{
		PortName:              "/dev/ttyS3",
		BaudRate:              115200,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       1,
		InterCharacterTimeout: 200,
	}

	p, err := serial.Open(options)
	if err != nil {
		return err
	}

	defer p.Close()

	testString := "hi there"
	n, err := p.Write([]byte(testString))
	if err != nil {
		return err
	}

	if n != len(testString) {
		return errors.New("# char written is wrong")
	}

	readString := make([]byte, 100)

	n, err = p.Read(readString)

	if err != nil {
		return err
	}

	readString = readString[:n]

	if testString != string(readString) {
		fmt.Println("read data: ", string(readString))
		return errors.New("read string failed")
	}

	return nil
}

type rs485 struct{}

func (d rs485) String() string {
	return "rs485"
}

func (d rs485) Run() error {
	GetInput("connect rs485 adapter to IS main pins #7 and #8")
	isio.GpioOut(isio.GpioSerialShutdown, false)
	isio.GpioOut(isio.GpioSerialLoopback, false)
	isio.GpioOut(isio.GpioSerialRsSelectRs485, true)

	options := serial.OpenOptions{
		PortName:               "/dev/ttyS3",
		BaudRate:               115200,
		DataBits:               8,
		StopBits:               1,
		MinimumReadSize:        1,
		InterCharacterTimeout:  200,
		Rs485Enable:            true,
		Rs485RtsHighDuringSend: true,
	}

	isPort, err := serial.Open(options)
	if err != nil {
		return err
	}

	defer isPort.Close()

	options.PortName = "/dev/ttyUSB0"
	options.Rs485Enable = false
	options.Rs485RtsHighDuringSend = false

	testPort, err := serial.Open(options)
	if err != nil {
		return err
	}

	defer testPort.Close()

	readBuffer := make([]byte, 100)

	// first write to test port and read from IS port
	writeTest := "write to test port"
	n, err := testPort.Write([]byte(writeTest))
	if err != nil {
		return err
	}

	time.Sleep(200 * time.Millisecond)

	if n != len(writeTest) {
		return errors.New("Did not write all bytes to test port")
	}

	n, err = isPort.Read(readBuffer)
	readBuffer = readBuffer[:n]

	if writeTest != string(readBuffer) {
		fmt.Println("read data: ", string(readBuffer))
		return errors.New("Error wrong data from IS port")
	}

	readBuffer = make([]byte, 100)

	// first write to test port and read from IS port
	writeIs := "write to IS port"
	n, err = isPort.Write([]byte(writeIs))
	if err != nil {
		return err
	}

	time.Sleep(200 * time.Millisecond)

	if n != len(writeIs) {
		return errors.New("Did not write all bytes to IS port")
	}

	n, err = testPort.Read(readBuffer)
	readBuffer = readBuffer[:n]

	if writeIs != string(readBuffer) {
		fmt.Println("read data: ", string(readBuffer))
		return errors.New("Error wrong data from test port")
	}

	return nil
}

func init() {
	Register(rs232{})
	Register(rs485{})
}
