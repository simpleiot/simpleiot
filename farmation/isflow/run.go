package isflow

import (
	"encoding/binary"
	"fmt"
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
			byteSlice := make([]byte, 128)
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
				c, err := file.Read(byteSlice)
				if c%8 != 0 {
					fmt.Println("Warning, did not read multiple of 8 bytes from driver", c)
					continue
				}

				if err != nil {
					log.Println("Error reading gpio_edge_timer: ", err)
					continue
				}

				for i := 0; i < c; i += 8 {
					pulseCh <- edgeTsToTime(byteSlice[i : i+8])
				}

				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	simTicker := time.NewTicker(25 * time.Millisecond)
	if !sim {
		simTicker.Stop()
	}

	pulses := 0

	ticker := time.NewTicker(1000 * time.Millisecond)

	flowRateMovingAvg := movingaverage.New(config.FlowAvgWindow)

	resetFlowRateMovingAvg := func(win int) {
		flowRateMovingAvg = movingaverage.New(win)
	}

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
				if config.FlowAvgWindow != m.FlowAvgWindow {
					resetFlowRateMovingAvg(m.FlowAvgWindow)
				}
				config = m
			case data.Sample:
				switch m.Type {
				case isdata.SampleTypeSimFlowRate:
					dur := isdata.FlowToPulsePeriod(m.Value, config.PulsesPerGallon)
					if dur <= 0 {
						dur = 1000 * time.Hour
					}
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
				// we need send 4 samples:
				//  - inst flow over last 1 sec (include eng data such as avg,
				//    amount, pulses, and avg
				//  - moving window average over last X samples
				//  - amount

				// Calculate flow and amount
				sampleDuration := lastPulse.Sub(lastTick)
				flow := isdata.PulsesToFlow(lastPulse, sampleDuration, config.PulsesPerGallon, pulses)
				amountSample := data.Sample{
					Type:  isdata.SampleTypeAmount,
					Time:  lastPulse,
					Value: flow.Amount,
				}

				out <- amountSample

				flowRateMovingAvg.Add(flow.Rate)

				flow.RateAvg = flowRateMovingAvg.Avg()
				out <- flow

				// Instantaneous flow sample
				// this sample is used for logging engineering data
				avgFlowSample := data.Sample{
					Time:  lastPulse,
					Type:  isdata.SampleTypeFlowWindowAvg,
					Value: flow.RateAvg,
				}

				avgFlowSample.Min, _ = flowRateMovingAvg.Min()
				avgFlowSample.Max, _ = flowRateMovingAvg.Max()
				out <- avgFlowSample

				pulses = 0
				lastTick = lastPulse
			}

			if time.Now().Sub(lastTick) > time.Second*5 {
				flow := data.Sample{
					Type:  isdata.SampleTypeFlowWindowAvg,
					Time:  time.Now(),
					Value: 0,
				}
				out <- flow
				resetFlowRateMovingAvg(config.FlowAvgWindow)
			}

		case t := <-simTicker.C:
			processPulse(t)
		}
	}
}
