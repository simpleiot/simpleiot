package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/goburrow/modbus"
)

func usage() {
	fmt.Println("Usage: ")
	flag.PrintDefaults()
	os.Exit(-1)
}

func main() {
	log.Println("modbus client")

	port := flag.String("port", "", "serial port")
	flag.Parse()

	if *port == "" {
		usage()
	}

	handler := modbus.NewRTUClientHandler(*port)
	handler.BaudRate = 115200
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 0
	handler.SlaveId = 1
	handler.Timeout = 5 * time.Second

	err := handler.Connect()
	if err != nil {
		log.Printf("%v\n", err)
		return
	}
	defer handler.Close()
	client := modbus.NewClient(handler)

	// Read discrete inputs.
	results, err := client.ReadDiscreteInputs(0, 16)
	if err != nil {
		log.Printf("%v\n", err)
	}

	fmt.Printf("results %v\n", results)
}
