// Package main tests response reader close functionality.
package main

import (
	"log"
	"os"
	"time"

	"github.com/simpleiot/simpleiot/respreader"
	"go.bug.st/serial"
)

func main() {
	mode := &serial.Mode{
		BaudRate: 115200,
	}

	for {
		time.Sleep(time.Second * 3)
		log.Println("================================")
		port, err := serial.Open(os.Args[1], mode)
		if err != nil {
			log.Println("Error opening port")
			continue
		}

		log.Println("Port opened")

		r := respreader.NewReadWriteCloser(port, time.Second, time.Millisecond*10)

		time.Sleep(time.Second * 3)

		log.Println("Closing port")

		r.Close()
	}
}
