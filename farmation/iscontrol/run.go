package iscontrol

import (
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// ISControl is used to control relays, system arm, and send faults
type ISControl struct {
	config       *isdata.Config
	state        *isdata.State
	stateMachine *StateMachine
	out          chan interface{}
}

// NewISControl initializes a RelayControl struct with the necessary parameters
func NewISControl(config *isdata.Config, state *isdata.State, stateMachine *StateMachine, out chan interface{}) *ISControl {
	return &ISControl{
		config:       config,
		state:        state,
		stateMachine: stateMachine,
		out:          out,
	}
}

// Update checks if anything need updated, and sends commands to out
func (rc *ISControl) Update() {
	// boolean to set
	var b bool

	// set inj pump relay
	if rc.config.ManualRelayInj == isdata.RelayControlStateAuto {
		rc.out <- isdata.UpdateGpioRelayInjector(rc.stateMachine.RelayInjector)
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
	statusLed := NewStatusLed(&config, &state, stateMachine, out)
	isControl := NewISControl(&config, &state, stateMachine, out)

	updateTicker := time.NewTicker(500 * time.Millisecond)
	ledTicker := time.NewTicker(350 * time.Millisecond)

	for {
		select {
		case <-updateTicker.C:

			flowStatus := GetFlowStatus(&state, &config)
			if state.FlowStatus != flowStatus {
				out <- isdata.UpdateFlowStatus(flowStatus)
			}

			msgs := stateMachine.Run()
			for _, msg := range msgs {
				out <- msg
			}
			isControl.Update()

		case <-ledTicker.C:
			statusLed.UpdateLedAction()
		case m := <-in:
			switch m := m.(type) {
			case isdata.State:
				state = m
			case isdata.Config:
				config = m

				msg := stateMachine.Run()
				if msg != nil {
					out <- msg
				}
				isControl.Update()

			}
		}

	}
}
