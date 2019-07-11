package ispressure

import (
	"log"
	"runtime"
	"time"

	movingaverage "github.com/RobinUS2/golang-moving-average"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isio"
)

// Run goroutine for IO code
func Run(in, out chan interface{}, configIn isdata.Config) {
	config := configIn
	ref, sense, err := isio.ReadPressure()
	if err != nil {
		log.Println("ReadPressure error: ", err)
	}

	senseSamplePeriod := 10 * time.Millisecond
	refSamplePeriod := 10 * time.Second
	reportPeriod := time.Second

	senseTicker := time.NewTicker(senseSamplePeriod)
	refTicker := time.NewTicker(refSamplePeriod)
	reportTicker := time.NewTicker(reportPeriod)

	if runtime.GOARCH != "arm" {
		senseTicker.Stop()
		refTicker.Stop()
		reportTicker.Stop()
	}

	pressureMovingAvg := movingaverage.New(int(4 * time.Second / senseSamplePeriod))
	var avg, min, max float64

	for {
		select {
		case <-senseTicker.C:
			sense, err = isio.ReadPressureSense()
			if err != nil {
				log.Println("ReadPressureSense error: ", err)
			}
			pres := isio.CalcPressure(ref, sense, float64(config.PressureSetting))
			if pres < 0 {
				pres = 0
			}
			pressureMovingAvg.Add(pres)
			avg = pressureMovingAvg.Avg()
			min, _ = pressureMovingAvg.Min()
			max, _ = pressureMovingAvg.Max()
			out <- data.Sample{
				Type:  isdata.SampleTypePressure,
				Time:  time.Now(),
				Value: pres,
				Attributes: map[string]float64{
					"avg": avg,
					"min": min,
					"max": max,
				},
			}

		case <-refTicker.C:
			ref, sense, err = isio.ReadPressure()
			if err != nil {
				log.Println("ReadPressure error: ", err)
			}

		case <-reportTicker.C:
			t := time.Now()
			out <- data.Sample{
				Time:  t,
				Type:  isdata.SampleTypePressureMin,
				Value: min,
			}

			out <- data.Sample{
				Time:  t,
				Type:  isdata.SampleTypePressureMax,
				Value: max,
			}

			out <- data.Sample{
				Time:  t,
				Type:  isdata.SampleTypePressureAvg,
				Value: avg,
			}

			out <- data.Sample{
				Time:  t,
				Type:  isdata.SampleTypePressureVRef,
				Value: ref,
			}

			out <- data.Sample{
				Time:  t,
				Type:  isdata.SampleTypePressureVSense,
				Value: sense,
			}

		case m := <-in:
			switch m := m.(type) {
			case isdata.Config:
				config = m
			case data.Sample:
				switch m.Type {
				case isdata.SampleTypeSimPressure:
					t := time.Now()
					out <- data.Sample{
						Type:  isdata.SampleTypePressure,
						Time:  time.Now(),
						Value: m.Value,
						Attributes: map[string]float64{
							"avg": m.Value,
							"min": m.Value,
							"max": m.Value,
						},
					}
					out <- data.Sample{
						Time:  t,
						Type:  isdata.SampleTypePressureMax,
						Value: m.Value,
					}

					out <- data.Sample{
						Time:  t,
						Type:  isdata.SampleTypePressureAvg,
						Value: m.Value,
					}

					out <- data.Sample{
						Time:  t,
						Type:  isdata.SampleTypePressureMin,
						Value: m.Value,
					}

				default:
					log.Println("ispressure: Unhandled sample type: ", m.Type)
				}
			default:
				log.Printf("ispressure mux: unhandled message of type %T: %+v\r\n", m, m)

			}
		}
	}
}
