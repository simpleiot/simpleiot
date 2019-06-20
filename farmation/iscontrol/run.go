package iscontrol

import (
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// RelayControl is used to control relays
type RelayControl struct {
	config       *isdata.Config
	state        *isdata.State
	stateMachine *StateMachine
	out          chan interface{}
}

// NewRelayControl initializes a RelayControl struct with the necessary parameters
func NewRelayControl(config *isdata.Config, state *isdata.State, stateMachine *StateMachine, out chan interface{}) *RelayControl {
	return &RelayControl{
		config:       config,
		state:        state,
		stateMachine: stateMachine,
		out:          out,
	}
}

// Update checks if any relays need updated, and sends commands to out
func (rc *RelayControl) Update() {
	// boolean to set relays
	var b bool

	// set inj pump relay
	if rc.config.ManualRelayInj == isdata.RelayControlStateAuto {
		if rc.config.UserPumpMode == isdata.UserPumpModeAuto {
			// set Inj relay = Gpio Inj input
			rc.out <- isdata.UpdateGpioRelayInjector(rc.stateMachine.RelayInjector)
		} else {
			// set Inj relay off
			rc.out <- isdata.UpdateGpioRelayInjector(false)
			// TODO test mode
		}
	} else { // diag mode
		b = rc.config.ManualRelayInj.BoolVal()
		if rc.state.GpioRelayInjectorEn != b {
			rc.out <- isdata.UpdateGpioRelayInjector(b)
		}
	}

	// set shutdown relay
	if rc.config.ManualRelayShutdown == isdata.RelayControlStateAuto {
		rc.out <- isdata.UpdateGpioRelayShutdown(rc.stateMachine.RelayShutdown)
	} else { // diag mode
		b = rc.config.ManualRelayShutdown.BoolVal()
		if rc.state.GpioRelayShutdownEn != b {
			rc.out <- isdata.UpdateGpioRelayShutdown(b)
		}
	}

	shtdwn := rc.stateMachine.Shutdown
	if rc.state.IrrigationShutdown != shtdwn {
		rc.out <- isdata.UpdateIrrigationShutdown(shtdwn)
	}

	// set aux relay
	// diag mode
	b = rc.config.ManualRelayAux.BoolVal()
	if rc.state.GpioRelayAuxEn != b {
		rc.out <- isdata.UpdateGpioRelayAux(b)
	}
}

// Run goroutine for ui code
func Run(in, out chan interface{}, configInit isdata.Config, stateInit isdata.State) {
	config := configInit
	state := stateInit
	stateMachine := NewStateMachine(&config, &state)
	relayControl := NewRelayControl(&config, &state, stateMachine, out)

	updateTicker := time.NewTicker(500 * time.Millisecond)

	for {
		select {
		case <-updateTicker.C:

			// update flow status
			flowStatus := GetFlowStatus(&state, &config)
			if state.FlowStatus != flowStatus {
				out <- isdata.UpdateFlowStatus(flowStatus)
			}

			updateMsg := stateMachine.Run()
			relayControl.Update()
			if updateMsg != nil && config.Arm { // state machine only returning disarm command right now
				out <- updateMsg
			}

		case m := <-in:
			switch m := m.(type) {
			case isdata.State:
				state = m
			case isdata.Config:
				config = m
				updateMsg := stateMachine.Run()
				relayControl.Update()
				if updateMsg != nil && config.Arm { // state machine only returning disarm command right now
					out <- updateMsg
				}

			}
		}

	}
}
