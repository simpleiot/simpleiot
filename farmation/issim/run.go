package issim

import (
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/sim"
)

// Run goroutine for IO code
func Run(in, out chan interface{}) {
	c := time.Tick(1 * time.Second)
	flowSim := sim.NewSim(20, 1, 20, 50)
	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case data.Sample:
				// ... todo
				_ = m
			}
		case <-c:
			v := flowSim.Sim()
			out <- data.NewSample("", isdata.SampleTypeFlowRate, v)
		}
	}
}
