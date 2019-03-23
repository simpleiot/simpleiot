package main

import (
	"log"
	"os"
	"syscall"
)

func main() {
	spi, err := syscall.Open("/dev/spidev1.0", os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal("Error opening spi device: ", err)
	}

	b := make([]byte, 1)
	for i := 0; i < 4; i++ {
		b[0] = byte(i)
		n, err := syscall.Write(spi, b)
		if err != nil {
			log.Fatal("Error writing to spi port: ", err)
		}
		log.Println("Wrote bytes: ", n)
	}
}
