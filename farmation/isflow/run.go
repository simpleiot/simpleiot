package isflow

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

var pulsesPerGal = 3785

// Run goroutine for IO code
func Run(in, out chan interface{}) {
	pulseCh := make(chan bool)
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
				pulseCh <- true
			}
		}
	}()

	pulses := 0

	tickerPeriod := 2 * time.Second
	ticker := time.NewTicker(tickerPeriod)

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			default:
				log.Printf("isflow mux: unhandled message of type %T: %+v\r\n", m, m)

			}
		case <-pulseCh:
			pulses++
		case <-ticker.C:
			out <- isdata.PulsesToFlow(tickerPeriod, pulsesPerGal, pulses)
			pulses = 0
		}
	}
}
