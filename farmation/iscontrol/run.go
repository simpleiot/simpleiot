package iscontrol

import (
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// RelayControl is used to control relays
type RelayControl struct {
	config       *isdata.Config
	state        *isdata.Config
	stateMachine *StateMachine
	out          chan interface{}
}

// Update checks if any relays need updated, and sends commands to out
func (rc *RelayControl) Update() {
}

func updateRelays(config *isdata.Config, state *isdata.State, out chan interface{}) {
	// boolean to set relays
	var b bool

	// set inj pump relay
	if config.ManualRelayInj == isdata.RelayControlStateAuto {
		if config.UserPumpMode == isdata.UserPumpModeAuto {
			// set Inj relay = Gpio Inj input
			out <- isdata.UpdateGpioRelayInjector(state.GpioDigitalInjector)
		} else {
			// set Inj relay off
			out <- isdata.UpdateGpioRelayInjector(false)
			// TODO test mode
		}
	} else {
		b = config.ManualRelayInj.BoolVal()
		if state.GpioRelayInjectorEn != b {
			out <- isdata.UpdateGpioRelayInjector(b)
		}
	}

	// set shutdown relay
	b = config.ManualRelayShutdown.BoolVal()
	if state.GpioRelayShutdownEn != b {
		out <- isdata.UpdateGpioRelayShutdown(b)
	}

	// set aux relay
	b = config.ManualRelayAux.BoolVal()
	if state.GpioRelayAuxEn != b {
		out <- isdata.UpdateGpioRelayAux(b)
	}
}

// Run goroutine for ui code
func Run(in, out chan interface{}, configInit isdata.Config, stateInit isdata.State) {
	config := configInit
	state := stateInit
	updateTicker := time.NewTicker(500 * time.Millisecond)
	stateMachine := NewStateMachine(&config, &state)

	for {
		select {
		case <-updateTicker.C:

			// update flow status
			flowStatus := GetFlowStatus(&state, &config)
			if state.FlowStatus != flowStatus {
				out <- isdata.UpdateFlowStatus(flowStatus)
			}

			stateMachine.Run()
			updateRelays(&config, &state, out)

		case m := <-in:
			switch m := m.(type) {
			case isdata.State:
				state = m
			case isdata.Config:
				config = m
				stateMachine.Run()
				updateRelays(&config, &state, out)
			}
		}

	}
}
