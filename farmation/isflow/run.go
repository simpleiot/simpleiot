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
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

func edgeTsToTime(data []byte) time.Time {
	tSec := binary.LittleEndian.Uint32(data[0:4])
	tNsec := binary.LittleEndian.Uint32(data[4:8])
	return time.Unix(int64(tSec), int64(tNsec))
}

// Run goroutine for IO code
func Run(in, out chan interface{}, sim bool, configInit isdata.Config) {
	config := configInit
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

	simTicker := time.NewTicker(25 * time.Millisecond)
	if !sim {
		simTicker.Stop()
	}

	pulses := 0

	tickerPeriod := 1000 * time.Millisecond
	ticker := time.NewTicker(tickerPeriod)

	flowRateMovingAvg := movingaverage.New(30)
	var lastTick time.Time
	var lastPulse time.Time

	processPulse := func(t time.Time) {
		pulses++
		lastPulse = t
		if config.LogPulseData {
			out <- isdata.Pulse(t)
		}
	}

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.Config:
				config = m
			case data.Sample:
				switch m.Type {
				case isdata.SampleTypeSimFlowRate:
					dur := isdata.FlowToPulsePeriod(m.Value, config.PulsesPerGallon)
					if dur < 5*time.Millisecond {
						dur = 25 * time.Millisecond
					}
					simTicker = time.NewTicker(dur)
				}
			default:
				log.Printf("isflow mux: unhandled message of type %T: %+v\r\n", m, m)

			}
		case t := <-pulseCh:
			processPulse(t)
		case <-ticker.C:
			if pulses > 0 {
				sampleDuration := lastPulse.Sub(lastTick)
				flow := isdata.PulsesToFlow(lastPulse, sampleDuration, config.PulsesPerGallon, pulses)
				flowRateMovingAvg.Add(flow.Rate)
				flow.RateAvg = flowRateMovingAvg.Avg()
				flow.RateMin, _ = flowRateMovingAvg.Min()
				flow.RateMax, _ = flowRateMovingAvg.Max()
				out <- flow
				pulses = 0
				lastTick = lastPulse
			}

		case t := <-simTicker.C:
			processPulse(t)
		}
	}
}
