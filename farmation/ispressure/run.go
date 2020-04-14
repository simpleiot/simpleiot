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

	refReader := isio.NewAdcReader(isio.AdcPressureRef)
	senseReader := isio.NewAdcReader(isio.AdcPressureSense)

	var ref, sense float64
	ref = 1.0

	if runtime.GOARCH == "arm" {
		var err error
		ref, err = refReader.Read()
		if ref == 0 {
			ref = 0.0001
		}
		ref = ref / isio.PressureScale
		if err != nil {
			log.Println("Error reading ref: ", err)
			// avoid divide by 0 errors
			ref = 1.0
		}
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
			var err error
			sense, err = senseReader.Read()
			sense = sense / isio.PressureScale
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
				Min:   min,
				Max:   max,
			}

		case <-refTicker.C:
			var err error
			ref, err = refReader.Read()
			if ref == 0 {
				ref = 0.0001
			}
			ref = ref / isio.PressureScale
			if err != nil {
				log.Println("Read ref error: ", err)
			}

		case <-reportTicker.C:
			t := time.Now()
			out <- data.Sample{
				Time:  t,
				Type:  isdata.SampleTypePressure,
				Value: avg,
				Min:   min,
				Max:   max,
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
					out <- data.Sample{
						Type:  isdata.SampleTypePressure,
						Time:  time.Now(),
						Value: m.Value,
						Min:   m.Value,
						Max:   m.Value,
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
