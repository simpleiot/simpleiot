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
	timeEvent        time.Time // timestamp to determine if irrigator has been off for more than allowed time

	// state machine static outputs
	// always update method setFalse when adding new field here!!!
	RelayShutdown   bool
	RelayInjector   bool
	Arm             bool
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
	monitoringFlow
	flowOffTarget
	irrigatorOff
	shutdown1
	shutdownMonitor1
	shutdown2
	shutdownMonitor2
	shutdownDialog
	shutdownDialogAck
	disarm
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
	case flowOffTarget:
		return "flowOffTarget"
	case irrigatorOff:
		return "irrigatorOff"
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

// this method sets all the boolean outputs of the state machine false
// since that is the most common state
func (sm *StateMachine) setFalse() {
	sm.RelayShutdown = false
	sm.RelayInjector = false
	sm.Arm = false
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
	if sm.inMonitorShutdownStates() {
		if sm.config.OperatingMode != isdata.ISOperatingModeMonitorAndShutdown {
			sm.setState(monitorOnly)
			return nil
		}
	}

	switch sm.machineState {
	case monitorOnly:
		sm.RelayShutdown = false
		sm.RelayInjector = sm.state.GpioDigitalInjector
		sm.CurrentLedState = LedGreenBlnk

		if sm.config.OperatingMode == isdata.ISOperatingModeMonitorAndShutdown {
			sm.setState(standby)
		}

		if sm.state.DialogStateMachine.Active {
			return isdata.UpdateDialogClose{}
		}

	// below states are for monitor/shutdown
	case standby:
		sm.setFalse()
		sm.CurrentLedState = LedGreenBlnk

		if sm.config.Arm {
			if sm.state.GpioDigitalWaterOn {
				sm.setState(monitoringFlow)
			} else {
				sm.setState(waitingForWater)
			}
		}

	case waitingForWater:
		sm.setFalse()
		sm.Arm = true
		sm.CurrentLedState = LedGreen

		if sm.state.GpioDigitalWaterOn {
			sm.setState(monitoringFlow)
		} else {
			if !sm.state.DialogStateMachine.Active {
				sm.setState(waitingForWaterAck)
				return isdata.UpdateDialogMessage("Waiting for water")
			}
		}

	case waitingForWaterAck:
		sm.setFalse()
		sm.Arm = true
		sm.CurrentLedState = LedGreen

		if sm.state.DialogStateMachine.Active {
			if sm.state.DialogStateMachine.Acknowledged ||
				sm.state.GpioDigitalWaterOn {
				return isdata.UpdateDialogClose{}
			}
		} else {
			if sm.state.GpioDigitalWaterOn {
				sm.setState(monitoringFlow)
			}
		}

	case monitoringFlow:
		sm.setFalse()
		sm.RelayInjector = sm.state.GpioDigitalInjector
		sm.Arm = true
		sm.CurrentLedState = LedGreen

		if !sm.state.GpioDigitalWaterOn {
			sm.setState(waitingForWater)
		}

		if sm.state.FlowStatus == isdata.FlowStatusOffTarget {
			sm.setState(flowOffTarget)
		}

		if sm.state.DialogStateMachine.Active {
			return isdata.UpdateDialogClose{}
		}

		if sm.state.GpioDigitalIrrigator {
			sm.timeEvent = time.Now()
		}

		// if alarm time has elapsed
		if time.Since(sm.timeEvent) >= time.Duration(sm.config.IrrigatorOffMin)*time.Minute {
			sm.setState(irrigatorOff)
		}

	case flowOffTarget:
		sm.setFalse()
		sm.RelayInjector = sm.state.GpioDigitalInjector
		sm.Arm = true
		sm.CurrentLedState = LedRedBlnk

		// if alarm time has elapsed enter shutdown
		if sm.elapsed() >= time.Duration(sm.config.AlarmRecognizeSec)*time.Second {
			sm.setState(disarm)
		} else if sm.state.FlowStatus == isdata.FlowStatusArmedOk {
			sm.setState(monitoringFlow)
		}

	case irrigatorOff:
		sm.setFalse()
		sm.setState(disarm)
		return isdata.UpdateFaultIrrigator(true)

	case disarm:
		sm.setFalse()
		sm.CurrentLedState = LedRed
		sm.setState(shutdown1)

	case shutdown1:
		sm.setFalse()
		sm.RelayShutdown = true
		sm.CurrentLedState = LedRed

		if sm.elapsed() > 10*time.Second {
			sm.setState(shutdownMonitor1)
		}

	case shutdownMonitor1:
		sm.setFalse()
		sm.CurrentLedState = LedRed

		if sm.elapsed() > 10*time.Second {
			if sm.state.GpioDigitalWaterOn {
				sm.setState(shutdown2)
			} else {
				sm.setState(shutdownDialog)
			}
		}

	case shutdown2:
		sm.setFalse()
		sm.RelayShutdown = true
		sm.CurrentLedState = LedRed

		if sm.elapsed() > 10*time.Second {
			sm.setState(shutdownMonitor2)
		}

	case shutdownMonitor2:
		sm.setFalse()
		sm.CurrentLedState = LedRed

		if sm.elapsed() > 10*time.Second {
			sm.setState(shutdownDialog)
		}

	case shutdownDialog:
		sm.setFalse()
		sm.CurrentLedState = LedRed

		if !sm.state.DialogStateMachine.Active {
			sm.setState(standby)
			if sm.state.GpioDigitalWaterOn {
				return isdata.UpdateDialogMessage("failed to shutdown")
			}

			return isdata.UpdateDialogMessage("system shut down")
		}
	}

	return nil
}
