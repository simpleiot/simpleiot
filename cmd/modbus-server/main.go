package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/modbus"
	"github.com/simpleiot/simpleiot/respreader"
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
	flagBaud := flag.String("baud", "9600", "baud rate")

	flag.Parse()

	if *flagPort == "" {
		usage()
	}

	baud, err := strconv.Atoi(*flagBaud)

	if err != nil {
		log.Println("Baud rate error: ", err)
		os.Exit(-1)
	}

	log.Printf("Starting server on: %v, baud: %v", *flagPort, baud)

	mode := &serial.Mode{
		BaudRate: baud,
	}
	port, err := serial.Open(*flagPort, mode)
	if err != nil {
		log.Fatal(err)
	}

	portRR := respreader.NewReadWriteCloser(port, time.Second, time.Millisecond*30)

	transport := modbus.NewRTU(portRR)

	serv := modbus.NewServer(1, transport)
	serv.Regs.AddCoil(128)
	err = serv.Regs.WriteCoil(128, true)
	if err != nil {
		log.Println("Error writing coil: ", err)
		os.Exit(-1)
	}

	serv.Regs.AddReg(2, 1)
	err = serv.Regs.WriteReg(2, 5)
	if err != nil {
		log.Println("Error writing reg: ", err)
		os.Exit(-1)
	}

	// start slave so it can respond to requests
	go serv.Listen(1, func(err error) {
		log.Println("modbus server listen error: ", err)
	}, func() {
		log.Printf("modbus reg changes")
	})

	if err != nil {
		log.Println("Error opening modbus port: ", err)
	}

	value := true
	regValue := 0
	up := true

	for {
		time.Sleep(time.Second * 10)

		value = !value
		serv.Regs.WriteCoil(128, value)

		if up {
			regValue = regValue + 1
			if regValue >= 10 {
				up = false
			}
		} else {
			regValue = regValue - 1
			if regValue <= 0 {
				up = true
			}
		}
		serv.Regs.WriteReg(2, uint16(regValue))
	}
}
