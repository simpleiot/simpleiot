package ispressure

import (
	"log"
	"runtime"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isio"
)

func pressure(out chan interface{}) {
	ref, _, err := isio.ReadPressure()
	if err != nil {
		log.Println("Error reading pressure: ", err)
		return
	}
	for {
		sense, err := isio.ReadPressureSense()
		if err != nil {
			log.Println("Error reading pressure sense: ", err)
			time.Sleep(30 * time.Second)
			continue
		}

		pres := isio.CalcPressure(ref, sense, 250)

		out <- data.Sample{
			Time:  time.Now(),
			Value: pres,
		}
	}
}

// Run goroutine for IO code
func Run(in, out chan interface{}) {
	if runtime.GOARCH == "arm" {
		go pressure(out)
	}
	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case data.Sample:
				// ... todo
				_ = m
			}
		}
	}
}
