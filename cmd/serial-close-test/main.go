package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/cbrake/go-serial/serial"
)

// the below test illustrates out the goroutine in the reader will close if you close
// the underlying serial port descriptor
func main() {
	log.Println("=============================")
	log.Println("Testing serial port close")

	if len(os.Args) < 2 {
		log.Println("Usage: serial-close-test /dev/ttyUSBx")
		os.Exit(-1)
	}

	port := os.Args[1]

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
		log.Fatal("Error opening serial port: ", err)
	}

	var wg sync.WaitGroup

	go func() {
		wg.Add(1)
		rdata := make([]byte, 128)
		for {
			log.Println("calling read on serial port")
			c, err := fread.Read(rdata)
			if err == io.EOF {
				log.Println("serial read returned EOF, yeah, this is good!")
				break
			}
			if err != nil {
				log.Println("serial read error: ", err)
			}
			log.Println("read count: ", c)
		}
		wg.Done()
	}()

	// Fifo test
	os.Remove("fifo")
	err = exec.Command("mkfifo", "fifo").Run()
	if err != nil {
		log.Fatal("mkfifo failed: ", err)
	}

	var fFifoRead io.ReadWriteCloser

	go func() {
		wg.Add(1)
		fFifoRead, err = os.OpenFile("fifo", os.O_RDONLY, 0600)
		if err != nil {
			log.Fatal("Error opening fifo for read: ", err)
		}

		rdata := make([]byte, 128)
		for {
			log.Println("call read on fifo")
			c, err := fFifoRead.Read(rdata)
			log.Printf("read %v bytes from fifo", c)
			if err == io.EOF {
				log.Println("fifo read error: ", err)
				break
			}

			if c == 0 {
				log.Println("read 0 bytes from fifo, done")
				break
			}
		}
		wg.Done()
	}()

	time.Sleep(500 * time.Millisecond)

	fFifoWrite, err := os.OpenFile("fifo", os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Error opening fifo for write: ", err)
	}
	fFifoWrite.Write([]byte("hi there"))

	// the following should unblock the above read
	// but it does not
	log.Println("Closing serial and fifo read files")
	fread.Close()
	fFifoRead.Close()

	log.Println("Waiting for reads to unblock ....")
	wg.Wait()

	log.Println("test all done, both reads were unblocked")
}
