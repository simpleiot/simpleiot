package isui

import (
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StatusLed is a type that stores all the states of the status led
type StatusLed struct {
	state   *isdata.State
	config  *isdata.Config
	redOn   bool
	greenOn bool
	out     chan interface{}
}

// NewStatusLed initializes the led state struct passing in state, config, and out channel
func NewStatusLed(state *isdata.State, config *isdata.Config, out chan interface{}) *StatusLed {
	return &StatusLed{
		state:  state,
		config: config,
		out:    out,
	}
}

// UpdateLedState updates the current led state based on config and state
func (sl *StatusLed) UpdateLedState() {
	switch {
	case sl.state.IrrigationShutdown: // shut down
		if sl.greenOn {
			sl.greenOn = false
			sl.out <- isdata.UpdateLedGreen(sl.greenOn)
		}
		sl.out <- isdata.UpdateLedRed(true) // solid red
		sl.redOn = true
	case sl.state.FlowStatus == isdata.FlowStatusOffTarget || len(sl.state.ActiveFaults) != 0: // flow off target or active faults
		if sl.greenOn {
			sl.greenOn = false
			sl.out <- isdata.UpdateLedGreen(sl.greenOn)
		}
		sl.redOn = !sl.redOn                    // blinking red
		sl.out <- isdata.UpdateLedRed(sl.redOn) //
	case sl.config.Arm: // armed
		if sl.redOn {
			sl.redOn = false
			sl.out <- isdata.UpdateLedRed(sl.redOn)
		}
		sl.out <- isdata.UpdateLedGreen(true) // solid green
		sl.greenOn = true
	default:
		if sl.redOn {
			sl.redOn = false
			sl.out <- isdata.UpdateLedRed(sl.redOn)
		}
		sl.greenOn = !sl.greenOn                    // blinking green
		sl.out <- isdata.UpdateLedGreen(sl.greenOn) //
	}

}
