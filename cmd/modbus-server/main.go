package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/simpleiot/mbserver"
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

	flag.Parse()

	if *flagPort == "" {
		usage()
	}

	log.Println("Starting server on: ", *flagPort)

	serv := mbserver.NewServer(50, 50, 50, 50)
	serv.Debug = true

	// set of serial port using respreader to do framing
	mode := &serial.Mode{
		BaudRate: 115200,
	}

	port, err := serial.Open(*flagPort, mode)
	if err != nil {
		log.Fatal(err)
	}

	portRR := respreader.NewResponseReadWriteCloser(port, time.Second, time.Millisecond*50)

	err = serv.ListenRTU(portRR)

	if err != nil {
		log.Println("Error opening modbus port: ", err)
	}

	defer serv.Close()

	select {}

}
