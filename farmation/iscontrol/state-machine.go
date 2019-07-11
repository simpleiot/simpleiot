package iscontrol

import (
	"log"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StateMachine ..
type StateMachine struct {
	// state machine inputs
	config *isdata.Config
	state  *isdata.State

	// state machine internals
	machineState     state
	timeStateEntered time.Time
	lastGoodFlow     time.Time

	// state machine static outputs
	RelayShutdown   bool
	RelayInjector   bool
	CurrentLedState LedState
}

// LedState is the state of the status LED
type LedState int

// define valid status LED states
const (
	LedGreenBlnk LedState = iota
	LedGreen
	LedRedBlnk
	LedRed
)

// State of machine
type state int

// define valid states
const (
	// states for monitor only
	monitorOnly state = iota

	// states for monitor/shutdown
	// the Start/End markers are not actually used
	// but allow us to easily detect this group of states
	monitorShutdownStart
	standby
	waitingForWater
	waitingForWaterAck
	waitingForIrr
	waitingForIrrAck
	monitoringFlow
	shutdownStart
	disarm
	shutdown1
	shutdownMonitor1
	shutdown2
	shutdownMonitor2
	shutdownDialog
	shutdownDialogAck
	monitorShutdownEnd

	// states for monitor/batch
)

func (s state) String() string {
	switch s {
	case monitorOnly:
		return "monitorOnly"
	case standby:
		return "standby"
	case monitoringFlow:
		return "monitoringFlow"
	case waitingForWater:
		return "waitingForWater"
	case waitingForWaterAck:
		return "waitingForWaterAck"
	case waitingForIrr:
		return "waitingForIrr"
	case waitingForIrrAck:
		return "waitingForIrrAck"
	case shutdown1:
		return "shutdown1"
	case shutdownMonitor1:
		return "shutdownMonitor1"
	case shutdown2:
		return "shutdown2"
	case shutdownMonitor2:
		return "shutdownMonitor2"
	case shutdownDialog:
		return "shutdownDialog"
	case shutdownDialogAck:
		return "shutdownDialogAck"
	case disarm:
		return "disarm"
	default:
		return strconv.Itoa(int(s))
	}
}

// NewStateMachine creates a new state machine
func NewStateMachine(config *isdata.Config, state *isdata.State) *StateMachine {
	return &StateMachine{
		config: config,
		state:  state,
	}
}

func (sm *StateMachine) setState(newState state) {
	if sm.machineState != newState {
		log.Println("New state: ", newState)
		sm.machineState = newState
		sm.timeStateEntered = time.Now()
	}
}

func (sm *StateMachine) elapsed() time.Duration {
	return time.Since(sm.timeStateEntered)
}

func (sm *StateMachine) inMonitorShutdownStates() bool {
	if sm.machineState >= monitorShutdownStart &&
		sm.machineState <= monitorShutdownEnd {
		return true
	}

	return false
}

func (sm *StateMachine) inShutdownStates() bool {
	if sm.machineState >= shutdownStart &&
		sm.machineState <= monitorShutdownEnd {
		return true
	}
	return false
}

// Run executes state machine, and returns an Update command if necessary.
// Rules for the state machine
//   - don't set state machine outputs in transition tests. The state outputs
//     should always be static during a state.
//   - generally all outputs should be set in each state -- this way we don't
//     forget something, and the output is correct no matter how the state is
//     entered, IE we're not depending on the output being set correctly in the
//     last state. TODO figure out some mechanism that forces this.
//   - if Update messages are returned from Run(), the state should continue to return
//     the message and only exit the state once the verified behavior has happened.
func (sm *StateMachine) Run() interface{} {

	// set relays to false and set to other values as needed in case statements
	sm.RelayShutdown = false
	sm.RelayInjector = false

	// before running state machine:
	if sm.inMonitorShutdownStates() {
		if sm.config.OperatingMode != isdata.ISOperatingModeMonitorAndShutdown {
			sm.setState(monitorOnly)
			return nil
		}
		// if disarmed in non-shutdown modes, go to standby
		if !sm.inShutdownStates() {
			if !sm.config.Arm {
				sm.setState(standby)
			}
		}
	}

	switch sm.machineState {
	case monitorOnly:

		sm.RelayInjector = sm.state.GpioDigitalInjector
		sm.CurrentLedState = LedGreenBlnk

		if sm.config.OperatingMode == isdata.ISOperatingModeMonitorAndShutdown {
			sm.setState(standby)
		}

		if sm.state.DialogStateMachine.Active {
			return isdata.UpdateDialogStateMachineClose{}
		}

	// below states are for monitor/shutdown
	case standby:

		sm.RelayInjector = sm.state.GpioDigitalInjector
		sm.CurrentLedState = LedGreenBlnk

		if sm.config.Arm {
			if sm.state.GpioDigitalWaterOn {
				sm.setState(monitoringFlow)
			} else {
				sm.setState(waitingForWater)
			}
		}

	case waitingForWater:

		sm.CurrentLedState = LedGreen

		if sm.state.GpioDigitalWaterOn {
			sm.setState(monitoringFlow)
		} else {
			if !sm.state.DialogStateMachine.Active {
				sm.setState(waitingForWaterAck)
				return isdata.UpdateDialogStateMachineMessage("Waiting for water")
			}
		}

	case waitingForWaterAck:

		sm.CurrentLedState = LedGreen

		if sm.state.DialogStateMachine.Active {
			if sm.state.DialogStateMachine.Acknowledged ||
				sm.state.GpioDigitalWaterOn {
				return isdata.UpdateDialogStateMachineClose{}
			}
		} else {
			if sm.state.GpioDigitalWaterOn {
				sm.setState(monitoringFlow)
			}
		}

	case waitingForIrr:

		sm.CurrentLedState = LedGreen

		if sm.state.GpioDigitalIrrigator {
			sm.setState(monitoringFlow)
		} else {
			if !sm.state.DialogStateMachine.Active {
				sm.setState(waitingForIrrAck)
				return isdata.UpdateDialogStateMachineMessage("Waiting for irrigator")
			}
		}

	case waitingForIrrAck:

		sm.CurrentLedState = LedGreen

		if sm.state.DialogStateMachine.Active {
			if sm.state.DialogStateMachine.Acknowledged ||
				sm.state.GpioDigitalIrrigator {
				return isdata.UpdateDialogStateMachineClose{}
			}
		} else {
			if sm.state.GpioDigitalIrrigator {
				sm.setState(monitoringFlow)
			}
		}

	case monitoringFlow:
		sm.RelayInjector = sm.state.GpioDigitalInjector

		if sm.state.FlowStatus == isdata.FlowStatusOffTarget {
			sm.CurrentLedState = LedRedBlnk
		} else {
			sm.CurrentLedState = LedGreen
		}

		if sm.state.DialogStateMachine.Active {
			return isdata.UpdateDialogStateMachineClose{}
		}

		if sm.state.FlowStatus == isdata.FlowStatusArmedOk {
			sm.lastGoodFlow = time.Now()
		}

		// the following switch statement is used only to determine next case. Keep all other logic
		// above.
		switch {
		case !sm.state.GpioDigitalWaterOn:
			sm.setState(waitingForWater)

		case !sm.state.GpioDigitalIrrigator:
			sm.setState(waitingForIrr)

		case time.Since(sm.lastGoodFlow) >= time.Duration(sm.config.AlarmRecognizeSec)*time.Second &&
			sm.state.GpioDigitalInjector:
			sm.setState(disarm)

		case sm.config.PressureShutdownEnabled &&
			sm.state.PressureMin < sm.config.PressureShutdownLow &&
			sm.state.GpioDigitalInjector:
			sm.setState(disarm)
			return isdata.UpdateFault{
				Fault: isdata.FaultTypeLowPres,
				Time:  time.Now(),
			}
		}

	case disarm:

		sm.CurrentLedState = LedRed

		if sm.config.Arm {
			return isdata.UpdateDisarm{}
		}
		sm.setState(shutdown1)

	case shutdown1:

		sm.RelayShutdown = true
		sm.CurrentLedState = LedRed

		if sm.elapsed() > 10*time.Second {
			sm.setState(shutdownMonitor1)
		}

	case shutdownMonitor1:

		sm.CurrentLedState = LedRed

		if sm.elapsed() > 10*time.Second {
			if sm.state.GpioDigitalWaterOn {
				sm.setState(shutdown2)
			} else {
				sm.setState(shutdownDialog)
			}
		}

	case shutdown2:

		sm.RelayShutdown = true
		sm.CurrentLedState = LedRed

		if sm.elapsed() > 10*time.Second {
			sm.setState(shutdownMonitor2)
		}

	case shutdownMonitor2:

		sm.CurrentLedState = LedRed

		if sm.elapsed() > 10*time.Second {
			sm.setState(shutdownDialog)
		}

	case shutdownDialog:

		sm.CurrentLedState = LedRed

		if !sm.state.DialogStateMachine.Active {
			sm.setState(standby)
			if sm.state.GpioDigitalWaterOn {
				return isdata.UpdateDialogStateMachineMessage("failed to shutdown")
			}

			return isdata.UpdateDialogStateMachineMessage("system shut down")
		}
	}

	return nil
}
