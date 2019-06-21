package iscontrol

import (
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StatusLed stores inputs and outputs of UpdateLedAction
type StatusLed struct {
	state        *isdata.State
	config       *isdata.Config
	stateMachine *StateMachine
	redOn        bool
	greenOn      bool
	out          chan interface{}
}

// NewStatusLed initializes the led state struct passing in state, config, and out channel
func NewStatusLed(config *isdata.Config, state *isdata.State, stateMachine *StateMachine, out chan interface{}) *StatusLed {
	return &StatusLed{
		state:        state,
		config:       config,
		stateMachine: stateMachine,
		out:          out,
	}
}

// UpdateLedAction updates the current led action based on the current led state
func (sl *StatusLed) UpdateLedAction() {
	switch {
	case sl.stateMachine.CurrentLedState == LedRed:
		if sl.greenOn {
			sl.greenOn = false
			sl.out <- isdata.UpdateLedGreen(sl.greenOn)
		}
		sl.out <- isdata.UpdateLedRed(true)
		sl.redOn = true
	case sl.stateMachine.CurrentLedState == LedRedBlnk:
		if sl.greenOn {
			sl.greenOn = false
			sl.out <- isdata.UpdateLedGreen(sl.greenOn)
		}
		sl.redOn = !sl.redOn
		sl.out <- isdata.UpdateLedRed(sl.redOn)
	case sl.stateMachine.CurrentLedState == LedGreen:
		if sl.redOn {
			sl.redOn = false
			sl.out <- isdata.UpdateLedRed(sl.redOn)
		}
		sl.out <- isdata.UpdateLedGreen(true)
		sl.greenOn = true
	case sl.stateMachine.CurrentLedState == LedGreenBlnk:
		if sl.redOn {
			sl.redOn = false
			sl.out <- isdata.UpdateLedRed(sl.redOn)
		}
		sl.greenOn = !sl.greenOn                    // blinking green
		sl.out <- isdata.UpdateLedGreen(sl.greenOn) //
	}

}
