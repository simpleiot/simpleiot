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
	lastGoodPressure time.Time

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
	standbyWaitingForWater
	standbyWaitingForWaterAck
	standbyWaitingForIrr
	standbyWaitingForIrrAck
	monitoringFlow
	monitorWaitingForWater
	monitorWaitingForWaterAck
	monitorWaitingForIrr
	monitorWaitingForIrrAck
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
	case standbyWaitingForWater:
		return "standbyWaitingForWater"
	case standbyWaitingForWaterAck:
		return "standbyWaitingForWaterAck"
	case standbyWaitingForIrr:
		return "standbyWaitingForIrr"
	case standbyWaitingForIrrAck:
		return "standbyWaitingForIrrAck"
	case monitoringFlow:
		return "monitoringFlow"
	case monitorWaitingForWater:
		return "monitorWaitingForWater"
	case monitorWaitingForWaterAck:
		return "monitorWaitingForWaterAck"
	case monitorWaitingForIrr:
		return "monitorWaitingForIrr"
	case monitorWaitingForIrrAck:
		return "monitorWaitingForIrrAck"
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

		if newState == monitoringFlow {
			// reset timestamps as we enter monitoring flow
			// so we give the flow and pressure time to stabalize
			sm.lastGoodFlow = time.Now()
			sm.lastGoodPressure = time.Now()
		}
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

func (sm *StateMachine) inStandbyWaitingStates() bool {
	if sm.machineState >= standbyWaitingForWater &&
		sm.machineState <= standbyWaitingForIrrAck {
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
		// if disarmed in non-shutdown and non-standbyWaiting states, go to standby
		if !sm.inShutdownStates() &&
			!sm.inStandbyWaitingStates() &&
			!sm.config.Arm {
			sm.setState(standby)
		}
	}

	switch sm.machineState {
	case monitorOnly:

		sm.RelayInjector = sm.state.InputInjector == isdata.InputStateOn
		sm.CurrentLedState = LedGreenBlnk

		if sm.config.OperatingMode == isdata.ISOperatingModeMonitorAndShutdown {
			sm.setState(standby)
		}

		if sm.state.DialogStateMachine.Active {
			return isdata.UpdateDialogStateMachineClose{}
		}

	// below states are for monitor/shutdown
	case standby:
		sm.RelayInjector = sm.state.InputInjector == isdata.InputStateOn
		sm.CurrentLedState = LedGreenBlnk

		switch {
		case sm.state.InputWaterOn == isdata.InputStateOff:
			sm.setState(standbyWaitingForWater)

		case sm.state.InputIrrigator == isdata.InputStateOff:
			sm.setState(standbyWaitingForIrr)

		case sm.config.Arm:
			sm.setState(monitoringFlow)
		}

	case standbyWaitingForWater:

		sm.CurrentLedState = LedGreenBlnk

		if sm.state.InputWaterOn != isdata.InputStateOff {
			sm.setState(standby)
		} else {
			if !sm.state.DialogStateMachine.Active {
				sm.setState(standbyWaitingForWaterAck)
				return isdata.UpdateDialogStateMachineMessage("Waiting for water")
			}
		}

	case standbyWaitingForWaterAck:

		sm.CurrentLedState = LedGreenBlnk

		if sm.state.DialogStateMachine.Active {
			if sm.state.DialogStateMachine.Acknowledged ||
				sm.state.InputWaterOn != isdata.InputStateOff {
				return isdata.UpdateDialogStateMachineClose{}
			}
		} else {
			if sm.state.InputWaterOn != isdata.InputStateOff {
				sm.setState(standby)
			}
		}

	case standbyWaitingForIrr:

		sm.CurrentLedState = LedGreenBlnk

		if sm.state.InputIrrigator != isdata.InputStateOff {
			sm.setState(standby)
		} else {
			if !sm.state.DialogStateMachine.Active {
				sm.setState(standbyWaitingForIrrAck)
				return isdata.UpdateDialogStateMachineMessage("Waiting for irrigator")
			}
		}

	case standbyWaitingForIrrAck:

		sm.CurrentLedState = LedGreenBlnk

		if sm.state.DialogStateMachine.Active {
			if sm.state.DialogStateMachine.Acknowledged ||
				sm.state.InputIrrigator != isdata.InputStateOff {
				return isdata.UpdateDialogStateMachineClose{}
			}
		} else {
			if sm.state.InputIrrigator != isdata.InputStateOff {
				sm.setState(standby)
			}
		}

	case monitoringFlow:
		sm.RelayInjector = sm.state.InputInjector == isdata.InputStateOn

		lowPressure := sm.state.PressureMin < sm.config.PressureShutdownLow

		if sm.state.InputInjector != isdata.InputStateOff &&
			(sm.state.FlowStatus == isdata.FlowStatusOffTarget ||
				(sm.config.PressureShutdownEnabled && lowPressure)) {
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

		if !lowPressure {
			sm.lastGoodPressure = time.Now()
		}

		alarmRecognizeDuration := time.Duration(sm.config.AlarmRecognizeSec) * time.Second

		// the following switch statement is used only to determine next case. Keep all other logic
		// above.
		switch {
		case sm.state.InputWaterOn == isdata.InputStateOff:
			sm.setState(monitorWaitingForWater)

		case sm.state.InputIrrigator == isdata.InputStateOff:
			sm.setState(monitorWaitingForIrr)

		case time.Since(sm.lastGoodFlow) >= alarmRecognizeDuration &&
			sm.state.InputInjector != isdata.InputStateOff:
			sm.setState(disarm)

		case sm.config.PressureShutdownEnabled &&
			lowPressure &&
			sm.state.InputInjector != isdata.InputStateOff &&
			time.Since(sm.lastGoodPressure) >= alarmRecognizeDuration:
			sm.setState(disarm)
			return isdata.UpdateFault{
				Fault: isdata.FaultTypeLowPres,
				Time:  time.Now(),
			}
		}
	case monitorWaitingForWater:

		sm.CurrentLedState = LedGreen

		if sm.state.InputWaterOn != isdata.InputStateOff {
			sm.setState(monitoringFlow)
		} else {
			if !sm.state.DialogStateMachine.Active {
				sm.setState(monitorWaitingForWaterAck)
				return isdata.UpdateDialogStateMachineMessage("Waiting for water")
			}
		}

	case monitorWaitingForWaterAck:

		sm.CurrentLedState = LedGreen

		if sm.state.DialogStateMachine.Active {
			if sm.state.DialogStateMachine.Acknowledged ||
				sm.state.InputWaterOn != isdata.InputStateOff {
				return isdata.UpdateDialogStateMachineClose{}
			}
		} else {
			if sm.state.InputWaterOn != isdata.InputStateOff {
				sm.setState(monitoringFlow)
			}
		}

	case monitorWaitingForIrr:

		sm.CurrentLedState = LedGreen

		if sm.state.InputIrrigator != isdata.InputStateOff {
			sm.setState(monitoringFlow)
		} else {
			if !sm.state.DialogStateMachine.Active {
				sm.setState(monitorWaitingForIrrAck)
				return isdata.UpdateDialogStateMachineMessage("Waiting for irrigator")
			}
		}

	case monitorWaitingForIrrAck:

		sm.CurrentLedState = LedGreen

		if sm.state.DialogStateMachine.Active {
			if sm.state.DialogStateMachine.Acknowledged ||
				sm.state.InputIrrigator != isdata.InputStateOff {
				return isdata.UpdateDialogStateMachineClose{}
			}
		} else {
			if sm.state.InputIrrigator != isdata.InputStateOff {
				sm.setState(monitoringFlow)
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
			if sm.state.InputWaterOn == isdata.InputStateOn {
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
			if sm.state.InputWaterOn == isdata.InputStateOn {
				return isdata.UpdateDialogStateMachineMessage("failed to shutdown")
			}

			return isdata.UpdateDialogStateMachineMessage("system shut down")
		}
	}

	return nil
}
