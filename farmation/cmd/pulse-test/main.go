package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

func main() {

	byteSlice := make([]byte, 8)

	// open file for reading
	file, err := os.Open("/dev/gpio_edge_timer")
	if err != nil {
		log.Fatal(err)
	}

	for {
		_, err := io.ReadFull(file, byteSlice)
		if err != nil {
			log.Fatal(err)
		}

		tSec := binary.LittleEndian.Uint32(byteSlice[0:4])
		tNsec := binary.LittleEndian.Uint32(byteSlice[4:8])
		timeCurrent := time.Unix(int64(tSec), int64(tNsec))
		fmt.Println(timeCurrent.UnixNano() / 1000)
	}
}
