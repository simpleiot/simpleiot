package ispressure

import (
	"log"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isio"
)

// Run goroutine for IO code
func Run(in, out chan interface{}) {
	ref, sense, err := isio.ReadPressure()
	if err != nil {
		log.Println("ReadPressure error: ", err)
	}
	senseTicker := time.NewTicker(10 * time.Millisecond)
	refTicker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-senseTicker.C:
			sense, err = isio.ReadPressureSense()
			if err != nil {
				log.Println("ReadPressureSense error: ", err)
			}
			pres := isio.CalcPressure(ref, sense, 250)
			out <- data.Sample{
				Type:  isdata.SampleTypePressure,
				Time:  time.Now(),
				Value: pres,
			}

		case <-refTicker.C:
			ref, sense, err = isio.ReadPressure()
			if err != nil {
				log.Println("ReadPressure error: ", err)
			}

		case m := <-in:
			switch m := m.(type) {
			case data.Sample:
				// ... todo
				_ = m
			}
		}
	}
}
