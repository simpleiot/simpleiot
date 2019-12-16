package isflow

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

func edgeTsToTime(data []byte) time.Time {
	tSec := binary.LittleEndian.Uint32(data[0:4])
	tNsec := binary.LittleEndian.Uint32(data[4:8])
	return time.Unix(int64(tSec), int64(tNsec))
}

// Run goroutine for flow calculation code
func Run(in, out chan interface{}, sim bool, configInit isdata.Config) {
	config := configInit
	pulseCh := make(chan time.Time)

	var fPulseOutputPeriod *os.File

	if runtime.GOARCH == "arm" {
		var err error
		fPulseOutputPeriod, err = os.OpenFile("/sys/devices/platform/gpio_et/output_pulse_timer_period_ns", os.O_WRONLY, 0644)
		if err != nil {
			log.Println("Error opening pulse output file: ", err)
		}

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

	fma := NewFlowMovAvg(config.FlowAvgWindowLong, config.FlowAvgWindow, config.FlowAvgPercDiff)

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
				// In case the user changes the averaging window or the percent diff
				if config.FlowAvgWindowLong != m.FlowAvgWindowLong {
					fma.ResetAvg(MovAvgLong, m.FlowAvgWindowLong)
				}
				if config.FlowAvgWindow != m.FlowAvgWindow {
					fma.ResetAvg(MovAvgShort, m.FlowAvgWindow)
				}
				if config.FlowAvgPercDiff != m.FlowAvgPercDiff {
					fma.UpdatePercentDiff(m.FlowAvgPercDiff)
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
				// we need to send the following samples:
				//  - inst flow over last 1 sec (include eng data such as avg,
				//    amount, pulses, and avg)
				//  - moving window average over last X samples
				//  - amount

				// Calculate flow and amount
				sampleDuration := lastPulse.Sub(lastTick)
				lastTick = lastPulse
				if sampleDuration > 3*time.Second {
					pulses = 0
					break
				}
				flow := isdata.PulsesToFlow(lastPulse, sampleDuration, config.PulsesPerGallon, pulses)

				// Amount
				amountSample := data.Sample{
					Type:  isdata.SampleTypeAmount,
					Time:  lastPulse,
					Value: flow.Amount,
				}

				out <- amountSample

				flow.RateAvg, flow.RateMin, flow.RateMax, _ = fma.AddDataPoint(flow.Rate)

				if fPulseOutputPeriod != nil {
					nsPerPulseOut := int64((1e9 * 3600) / (float64(config.PulseOutputK) * flow.RateAvg))
					_, err := fPulseOutputPeriod.WriteString(strconv.FormatInt(nsPerPulseOut, 10) + "\n")
					if err != nil {
						log.Println("Error writing pulse output: ", err)
					}
				}

				// Instantaneous flow sample
				// this sample is used for logging engineering data
				out <- flow

				// Flow rate. This sample is used to update the flow
				// rate stored in system state
				avgFlowSample := data.Sample{
					Time:  lastPulse,
					Type:  isdata.SampleTypeFlowWindowAvg,
					Value: flow.RateAvg,
					Min:   flow.RateMin,
					Max:   flow.RateMax,
				}

				out <- avgFlowSample

				pulses = 0
			}

			if time.Since(lastTick) > time.Second*5 {
				flow := data.Sample{
					Type:  isdata.SampleTypeFlowWindowAvg,
					Time:  time.Now(),
					Value: 0,
				}
				out <- flow
				resetFlowRateMovingAvg(config.FlowAvgWindow)
				if fPulseOutputPeriod != nil {
					_, err := fPulseOutputPeriod.WriteString("0\n")
					if err != nil {
						log.Println("Error writing pulse output: ", err)
					}
				}
			}

		case t := <-simTicker.C:
			processPulse(t)
		}
	}
}
