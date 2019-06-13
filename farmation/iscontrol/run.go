package iscontrol

import (
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Run goroutine for ui code
func Run(in, out chan interface{}, configInit isdata.Config, stateInit isdata.State) {
	config := configInit
	state := stateInit

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.State:
				state = m
				UpdateFlowStatus(&state, &config)
				out <- state
			case isdata.Config:
				config = m
				UpdateFlowStatus(&state, &config)
				out <- state
			}
		}

	}
}
