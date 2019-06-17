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
		BaudRate:               57600,
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

	if runtime.GOARCH == "arm" {
		go serialListener(out)
	}

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.Config:
				config = m
			default:
				log.Printf("isflow mux: unhandled message of type %T: %+v\r\n", m, m)

			}
		}
	}
}
