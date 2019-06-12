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
	Off
)

// ComputeLedState updates the current led state based on config and state
func (sl *StatusLed) ComputeLedState() *StatusLed {
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
		sl.LedState = LedBlnkGreen // blinking green
	}

	return &sl
}

// DisplayLed takes action based on the led state
func (sl *StatusLed) DisplayLed() {
	sl = sl.ComputeLedState()
	switch sl.LedState {
	case LedRed:
		isio.GpioOut(isio.GpioRed, true)
	case LedGreen:
	case LedBlnkRed:
		if isio.GpioRead(isio.GpioRed) {
			isio.GpioOut(isio.GpioRed, false)
		} else {
			isio.GpioOut(isio.GpioRed, true)
		}
	case LedBlnkGreen:

	}
}
