package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
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

	flag.Parse()

	if *flagPort == "" {
		usage()
	}

	mode := &serial.Mode{
		BaudRate: 115200,
	}

	port, err := serial.Open(*flagPort, mode)
	if err != nil {
		log.Fatal(err)
	}

	portRR := respreader.NewResponseReadWriteCloser(port, time.Second, time.Millisecond*50)

	for {
		data := make([]byte, 100)

		c, err := portRR.Read(data)

		if err != nil {
			if err != io.EOF {
				log.Println("Error reading serial port: ", err)
			}

			continue
		}

		if c <= 0 {
			continue
		}

		data = data[:c]

		fmt.Println("Received data: ", hex.Dump(data))

		err = modbus.CheckRtuCrc(data)

		if err != nil {
			log.Println("CRC error: ", err)
		}
	}
}
