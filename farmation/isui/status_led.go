package isui

import (
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StatusLed is a type that stores all the states of the status led
type StatusLed struct {
	LedState LedState
	state    *isdata.State
	config   *isdata.Config
}

// LedState holds the status led state
type LedState int

// Valid led states
const (
	LedRed LedState = iota
	LedGreen
	LedRedBlnk
	LedGreenBlnk
	LedOff
)

// NewStatusLed initializes the led state struct passing in state, config, and out channel
func NewStatusLed(state *isdata.State, config *isdata.Config) *StatusLed {
	return &StatusLed{
		state:  state,
		config: config,
	}
}

// ComputeLedState updates the current led state based on config and state
func (sl *StatusLed) ComputeLedState() {
	switch {
	case sl.config.Arm: // if IS is armed
		if sl.state.FlowStatus == isdata.FlowStatusOffTarget || len(sl.state.ActiveFaults) != 0 { // if flow rate off target or active faults
			sl.LedState = LedRedBlnk // blinking red
		} else {
			sl.LedState = LedGreen // solid green
		}
	case sl.state.IrrigationShutdown || len(sl.state.ActiveFaults) != 0: // if not armed and shutting down or active faults
		sl.LedState = LedRed // solid red
	default:
		sl.LedState = LedGreenBlnk // blinking green
	}

}
