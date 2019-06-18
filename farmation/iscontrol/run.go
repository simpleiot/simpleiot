package iscontrol

import (
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

func updateRelays(c *isdata.Config, s *isdata.State, out chan interface{}) {
	b := c.ManualRelayInj.BoolVal()
	if s.GpioRelayInjectorEn != b {
		out <- isdata.UpdateGpioRelayInj(b)
	}

	b = c.ManualRelayShutdown.BoolVal()
	if s.GpioRelayShutdownEn != b {
		out <- isdata.UpdateGpioRelayShutdown(b)
	}

	b = c.ManualRelayAux.BoolVal()
	if s.GpioRelayAuxEn != b {
		out <- isdata.UpdateGpioRelayAux(b)
	}
}

// Run goroutine for ui code
func Run(in, out chan interface{}, configInit isdata.Config, stateInit isdata.State) {
	config := configInit
	state := stateInit
	flowStatusUpdateTicker := time.NewTicker(1000 * time.Millisecond)

	for {
		select {
		case <-flowStatusUpdateTicker.C:
			flowStatus := GetFlowStatus(&state, &config)
			if state.FlowStatus != flowStatus {
				out <- isdata.UpdateFlowStatus(flowStatus)
			}

			updateRelays(&config, &state, out)

		case m := <-in:
			switch m := m.(type) {
			case isdata.State:
				state = m
			case isdata.Config:
				config = m
			}
		}

	}
}
