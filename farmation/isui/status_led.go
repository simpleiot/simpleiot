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

// UpdateLedState updates the current led state based on config and state
func (sl *StatusLed) UpdateLedState() {
	switch {
	case sl.state.IrrigationShutdown: // if not armed and shutting down or active faults
		sl.LedState = LedRed // solid red
	case sl.state.FlowStatus == isdata.FlowStatusOffTarget || len(sl.state.ActiveFaults) != 0:
		sl.LedState = LedRedBlnk
	case sl.config.Arm: // if IS is armed
		sl.LedState = LedGreen // solid green
	default:
		sl.LedState = LedGreenBlnk // blinking green
	}

}
