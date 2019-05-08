package isflow

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"runtime"
	"time"

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

	tickerPeriod := 2 * time.Second
	ticker := time.NewTicker(tickerPeriod)

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.Config:
				config = m
			default:
				log.Printf("isflow mux: unhandled message of type %T: %+v\r\n", m, m)

			}
		case pulse := <-pulseCh:
			pulses++
			if config.LogPulseData {
				out <- pulse
			}
		case <-ticker.C:
			out <- isdata.PulsesToFlow(tickerPeriod, pulsesPerGal, pulses)
			pulses = 0
		}
	}
}
