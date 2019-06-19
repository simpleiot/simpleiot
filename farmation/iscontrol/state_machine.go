package iscontrol

import (
	"log"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StateMachine ..
type StateMachine struct {
	config *isdata.Config
	state  *isdata.State

	// state machine internals
	machineState     state
	timeStateEntered time.Time

	// state machine static outputs (updated in StateMachine type)
	RelayShutdown   bool
	RelayInjector   bool
	FaultWaterNotOn bool

	// state machine command outputs (returned from Run() as update commands)
}

// State of machine
type state int

// define valid states
const (
	standby state = iota
	armed
	flowOffTarget
	shutdown1
	shutdownMonitor1
	shutdown2
	shutdownMonitor2
	waitForDisarm
)

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

// Run executes state machine, and returns an Update command if necessary
func (sm *StateMachine) Run() interface{} {

	// if user disarms, stop shutdown
	if !sm.config.Arm && sm.machineState != standby {
		sm.setState(standby)
		sm.RelayShutdown = false
	}

	switch sm.machineState {
	case standby:
		if sm.config.Arm {
			sm.setState(armed)
		}

	case armed:
		if sm.state.FlowStatus == isdata.FlowStatusOffTarget {
			sm.setState(flowOffTarget)
		}

	case flowOffTarget:
		// if alarm time has elapsed enter shutdown
		secondsSince := time.Since(sm.timeStateEntered).Seconds()
		if secondsSince >= sm.config.AlarmRecognizeSec {
			sm.setState(shutdown1)
		} else if sm.state.FlowStatus == isdata.FlowStatusArmedOk { // else if flow back in target, return to armed mode
			sm.setState(armed)
		}

	case shutdown1:
		sm.RelayShutdown = true
		sm.RelayInjector = false
		secondsSince := time.Since(sm.timeStateEntered).Seconds()
		if secondsSince >= 10 {
			sm.RelayShutdown = false
			sm.setState(shutdownMonitor1)
		}

	case shutdownMonitor1:
		if sm.state.GpioDigitalWaterOn { // if water is still on
			sm.setState(shutdown2)
		} else {
			// TODO wait for user acknowledge
			sm.setState(waitForDisarm)
			return isdata.UpdateDisarm(true)
		}

	case shutdown2:
		sm.RelayShutdown = true
		sm.RelayInjector = false
		secondsSince := time.Since(sm.timeStateEntered).Seconds()
		if secondsSince >= 10 {
			sm.RelayShutdown = false
			sm.setState(shutdownMonitor2)
		}

	case shutdownMonitor2:
		if sm.state.GpioDigitalWaterOn { // if water is still on
			// TODO display "irrigation system failed to shutdown"
			// TODO wait for user acknowledge
		} else {
			// TODO wait for user acknowledge
			sm.setState(waitForDisarm)
			return isdata.UpdateDisarm(true)
		}

	case waitForDisarm:
		secondsSince := time.Since(sm.timeStateEntered).Seconds()
		if sm.config.Arm {
			if secondsSince >= 10 {
				sm.setState(waitForDisarm)
				return isdata.UpdateDisarm(true)
			}
		} else {
			sm.setState(standby)
		}
	}

	return nil
}
