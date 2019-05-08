package issim

import (
	"time"

	"github.com/simpleiot/simpleiot/data"
)

// Run goroutine for IO code
func Run(in, out chan interface{}) {
	c := time.Tick(40 * time.Millisecond)
	//flowSim := sim.NewSim(20, 1, 20, 50)
	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case data.Sample:
				// ... todo
				_ = m
			}
		case <-c:
		}
	}
}
