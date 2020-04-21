package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/grid-x/modbus"
)

func usage() {
	fmt.Println("Usage: ")
	flag.PrintDefaults()
	os.Exit(-1)
}

func main() {
	log.Println("modbus client")

	flagPort := flag.String("port", "", "serial port")
	flag.Parse()

	if *flagPort == "" {
		usage()
	}

	handler := modbus.NewRTUClientHandler(*flagPort)
	handler.BaudRate = 115200
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SetSlave(1)
	handler.Timeout = 5 * time.Second

	err := handler.Connect()

	if err != nil {
		fmt.Println("Error connecting: ", err)
		os.Exit(-1)
	}
	defer handler.Close()

	client := modbus.NewClient(handler)
	results, err := client.ReadCoils(15, 1)

	fmt.Printf("results: %+v\n", results)
}
