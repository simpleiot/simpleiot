package isflow

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	movingaverage "github.com/RobinUS2/golang-moving-average"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

var pulsesPerGal = 3785

func edgeTsToTime(data []byte) time.Time {
	tSec := binary.LittleEndian.Uint32(data[0:4])
	tNsec := binary.LittleEndian.Uint32(data[4:8])
	return time.Unix(int64(tSec), int64(tNsec))
}

// Run goroutine for IO code
func Run(in, out chan interface{}, sim bool) {
	config := isdata.Config{}
	pulseCh := make(chan time.Time)
	if runtime.GOARCH == "arm" {
		go func() {
			// open file for reading
			byteSlice := make([]byte, 8)
			file, err := os.Open("/dev/gpio_edge_timer")
			if err != nil {
				log.Println("Error opening pulse meter driver: ", err)
				return
			}

			dumpBuffer := make([]byte, 100*1024)

			// dump any pulses that have accumlated in the driver so we don't
			// skew our results
			c, _ := file.Read(dumpBuffer)
			fmt.Println("Dumped pulse data: ", c)

			for {
				_, err := io.ReadFull(file, byteSlice)
				if err != nil {
					log.Println("Error reading gpio_edge_timer: ", err)
				} else {
					pulseCh <- edgeTsToTime(byteSlice)
				}
			}
		}()
	}

	if sim {
		go func() {
			simTicker := time.NewTicker(25 * time.Millisecond)
			for {
				select {
				case t := <-simTicker.C:
					pulseCh <- t
				}
			}
		}()
	}

	pulses := 0

	tickerPeriod := 1000 * time.Millisecond
	ticker := time.NewTicker(tickerPeriod)

	flowRateMovingAvg := movingaverage.New(30)

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.Config:
				config = m
			default:
				log.Printf("isflow mux: unhandled message of type %T: %+v\r\n", m, m)

			}
		case timeStamp := <-pulseCh:
			pulses++
			if config.LogPulseData {
				out <- isdata.Pulse(timeStamp)
			}
		case <-ticker.C:
			flow := isdata.PulsesToFlow(tickerPeriod, pulsesPerGal, pulses)
			flowRateMovingAvg.Add(flow.Rate)
			flow.RateAvg = flowRateMovingAvg.Avg()
			out <- flow
			pulses = 0
		}
	}
}
