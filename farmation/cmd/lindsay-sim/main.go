package main

import (
	"log"

	"github.com/cbrake/go-serial/serial"
	"github.com/simpleiot/simpleiot/farmation/isserial"
)

func main() {
	options := serial.OpenOptions{
		PortName:              "/dev/ttyUSB0",
		BaudRate:              57600,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       1,
		InterCharacterTimeout: 200,
	}

	port, err := serial.Open(options)
	if err != nil {
		log.Fatal("Error opening serial port: ", err)
	}

	defer port.Close()

	n, err := port.Write(isserial.LindsayTestData)

	if err != nil {
		log.Fatal("Error writing to port: ", err)
	}

	log.Println("Characters written: ", n)
}
