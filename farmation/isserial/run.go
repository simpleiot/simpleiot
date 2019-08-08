package isserial

import (
	"log"
	"runtime"
	"time"

	"github.com/cbrake/go-serial/serial"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isio"
)

func serialListener(out chan interface{}) {
	log.Println("Starting Lindsay serial protocol, rs485")
	isio.GpioOut(isio.GpioSerialShutdown, false)
	isio.GpioOut(isio.GpioSerialLoopback, false)
	isio.GpioOut(isio.GpioSerialRsSelectRs485, true)

	options := serial.OpenOptions{
		PortName:               isio.SerialRS232RS485,
		BaudRate:               19200,
		DataBits:               8,
		StopBits:               1,
		MinimumReadSize:        1,
		InterCharacterTimeout:  200,
		Rs485Enable:            true,
		Rs485RtsHighDuringSend: true,
	}

	isPort, err := serial.Open(options)
	if err != nil {
		log.Println("Error opening panel serial port: ", err)
		return
	}

	defer isPort.Close()

	lindsay := NewLindsay(isPort)

	for {
		status, err := lindsay.Read()
		if err == errorNotLindsayStatus {
			continue
		}
		if err != nil {
			log.Println("Error reading Lindsay port: ", err)
			// rate limit errors
			time.Sleep(10 * time.Second)
		}

		log.Println(status)
		out <- status
	}
}

// Run goroutine for IO code
func Run(in, out chan interface{}, configInit isdata.Config) {
	config := configInit
	_ = config

	serialTimeout := 30 * time.Second
	serialTimer := time.NewTimer(serialTimeout)

	if runtime.GOARCH != "arm" {
		serialTimer.Stop()
	}

	listChan := make(chan interface{}, 100)

	if runtime.GOARCH == "arm" {
		go serialListener(listChan)
	}

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.Config:
				config = m
			default:
				log.Printf("isserial mux: unhandled message of type %T: %+v\r\n", m, m)

			}
		case serialData := <-listChan:
			out <- serialData
			serialTimer.Reset(serialTimeout)

		case <-serialTimer.C:
			// timed out waiting for serial data so send 0 values
			out <- isdata.LindsayStatusRegs{}
		}
	}
}
