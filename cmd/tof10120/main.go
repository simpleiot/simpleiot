// TOF10120 test application
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/simpleiot/simpleiot/respreader"
	"github.com/simpleiot/simpleiot/sensors"
	"go.bug.st/serial"
)

func usage() {
	fmt.Println("Usage: ")
	flag.PrintDefaults()
	os.Exit(-1)
}

func main() {
	log.Println("modbus simulator")

	flagPort := flag.String("port", "", "serial port")
	flagSend := flag.Int("send", 200, "interval at which sensor should send data")

	flag.Parse()

	if *flagPort == "" {
		usage()
	}

	mode := &serial.Mode{
		BaudRate: 9600,
	}
	port, err := serial.Open(*flagPort, mode)
	if err != nil {
		log.Fatal(err)
	}

	portRR := respreader.NewReadWriteCloser(port, time.Second, time.Millisecond*20)

	tof := sensors.NewTOF10120(portRR)

	err = tof.SetSendInterval(*flagSend)

	if err != nil {
		log.Println("Error setting send interval:", err)
	}

	err = tof.Read(func(v int) {
		log.Printf("TOF data: %vmm\n", v)
	}, func(err error) {
		log.Println("Error reading TOF:", err)
	})

	if err != nil {
		log.Println("Error reading TOF:", err)
	}
}
