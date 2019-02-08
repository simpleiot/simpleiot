package main

import (
	"os"
	"log"
	"io"
	"encoding/binary"
	"fmt"
	"time"
//	"github.com/VividCortex/ewma"
	. "github.com/mxmCherry/movavg"
)

func main() {

	byteSlice := make([]byte, 8)
	sma := ThreadSafe(NewSMA(3))

	// open file for reading
	file, err := os.Open("/dev/gpio_edge_timer")
	if err != nil {
		log.Fatal(err)
	}

//	simple_ewma := ewma.NewMovingAverage()
//	variable_ewma := ewma.NewMovingAverage(5)

	timePrevious := time.Now()

	for {
	_, err := io.ReadFull(file, byteSlice)
	if err != nil {
		log.Fatal(err)
		}

	t_sec := binary.LittleEndian.Uint32(byteSlice[0:4])
	t_nsec := binary.LittleEndian.Uint32(byteSlice[4:8])
	timeCurrent := time.Unix(int64(t_sec), int64(t_nsec))
//	simple_ewma.Add((float64) (timeCurrent.Sub(timePrevious)))
//	variable_ewma.Add((float64) (timeCurrent.Sub(timePrevious)))
	sma.Add((float64) (timeCurrent.Sub(timePrevious)))
	fmt.Printf("sma(3): %v, raw: %d\n", sma.Avg(), timeCurrent.Sub(timePrevious))
	timePrevious = timeCurrent
	}
}
