package iscontrol

import (
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Run goroutine for ui code
func Run(in, out chan interface{}, configInit isdata.Config, stateInit isdata.State) {
	config := configInit
	state := stateInit
	updateTicker := time.NewTicker(1000 * time.Millisecond)

	for {
		select {
		case <-updateTicker.C:

			// update flow status
			flowStatus := GetFlowStatus(&state, &config)
			if state.FlowStatus != flowStatus {
				out <- isdata.UpdateFlowStatus(flowStatus)
			}

			// update state
			/*for _, updatemsg := range StateMachine.Run() { // extract update messages from slice returned by state machine
				out <- updatemsg // send each message to app.go
			}*/

			if config.ManualRelayInj == isdata.RelayControlStateAuto {
				// set inj pump relay
				if config.UserPumpMode == isdata.UserPumpModeAuto {
					// set Inj relay = Gpio Inj input
					out <- isdata.UpdateRelayInjector(state.GpioDigitalInjector)
				} else {
					// set Inj relay off
					out <- isdata.UpdateRelayInjector(false)
					// TODO test mode
				}
			} else {
				if config.ManualRelayInj == isdata.RelayControlStateOff {
					out <- isdata.UpdateRelayInjector(false)
				} else if config.ManualRelayInj == isdata.RelayControlStateOn {
					out <- isdata.UpdateRelayInjector(true)
				}
			}

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
