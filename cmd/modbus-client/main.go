// example modbus client application
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
	log.Println("modbus client")

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

	mode := &serial.Mode{
		BaudRate: baud,
	}

	port, err := serial.Open(*flagPort, mode)
	if err != nil {
		log.Fatal(err)
	}

	portRR := respreader.NewReadWriteCloser(port, time.Second*1, time.Millisecond*30)
	transport := modbus.NewRTU(portRR)
	client := modbus.NewClient(transport, 1)

	// Read discrete inputs.
	coils, err := client.ReadCoils(1, 128, 1)
	if len(coils) != 1 {
		log.Println("Error: Expected one coil result")
		os.Exit(-1)
	}

	log.Println("Coil results: ", coils)

	// read holding reg
	regs, err := client.ReadHoldingRegs(1, 2, 1)
	if len(regs) != 1 {
		log.Println("Error: Expected one reg result")
		os.Exit(-1)
	}

	log.Printf("Reg result: 0x%x\n", regs[0])
}
