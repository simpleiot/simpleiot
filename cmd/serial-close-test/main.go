package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cbrake/go-serial/serial"
)

// the below test illustrates out the goroutine in the reader will close if you close
// the underlying serial port descriptor
func main() {
	fmt.Println("=============================")
	fmt.Println("Testing serial port close")

	if len(os.Args) < 2 {
		fmt.Println("Usage: serial-close-test /dev/ttyUSBx")
		os.Exit(-1)
	}

	port := os.Args[1]

	done := make(chan bool)

	options := serial.OpenOptions{
		PortName:              port,
		BaudRate:              115200,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       1,
		InterCharacterTimeout: 0,
	}

	fread, err := serial.Open(options)
	if err != nil {
		fmt.Println("Error opening serial port: ", err)
		os.Exit(-1)
	}

	go func(readCnt chan bool) {
		rdata := make([]byte, 128)
		for {
			fmt.Println("calling read")
			c, err := fread.Read(rdata)
			if err == io.EOF {
				fmt.Println("Reader returned EOF, yeah, this is good!")
				break
			}
			if err != nil {
				fmt.Println("Read error: ", err)
			}
			fmt.Println("read count: ", c)
		}

		done <- true
	}(done)

	time.Sleep(500 * time.Millisecond)

	// the following should unblock the above read
	fmt.Println("Closing read file")
	fread.Close()

	<-done

	fmt.Println("test all done")
}
