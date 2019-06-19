package iscontrol

import (
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

	// state machine outputs
	RelayShutdown bool
	RelayInjector bool
	Armed         bool
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
)

// NewStateMachine creates a new state machine
func NewStateMachine(config *isdata.Config, state *isdata.State) *StateMachine {
	return &StateMachine{
		config: config,
		state:  state,
	}
}

// Run executes state machine
func (sm *StateMachine) Run() {
	switch sm.machineState {
	case standby:
		if sm.config.Arm {
			sm.machineState = armed
			sm.timeStateEntered = time.Now()
		}
	case armed:
		if sm.state.FlowStatus == isdata.FlowStatusOffTarget {
			sm.machineState = flowOffTarget
			sm.timeStateEntered = time.Now()
		}
	case flowOffTarget:

		// if alarm time has elapsed enter shutdown
		secondsSince := time.Since(sm.timeStateEntered).Seconds()
		//fmt.Println(secondsSince)
		if secondsSince >= sm.config.AlarmRecognizeSec {
			sm.machineState = shutdown1
			sm.timeStateEntered = time.Now()

		} else if sm.state.FlowStatus == isdata.FlowStatusArmedOk { // else if flow back in target, return to armed mode
			sm.machineState = armed
			sm.timeStateEntered = time.Now()
		}
	case shutdown1:
		sm.RelayShutdown = true
		sm.RelayInjector = false
		secondsSince := time.Since(sm.timeStateEntered).Seconds()
		if secondsSince >= 10 {
			sm.RelayShutdown = false
			sm.machineState = shutdownMonitor1
			sm.timeStateEntered = time.Now()
		}
	case shutdownMonitor1:
		if sm.state.GpioDigitalWaterOn { // if water is still on
			sm.machineState = shutdown2
			sm.timeStateEntered = time.Now()
		} else {
			// TODO wait for user acknowledge
			sm.Armed = false
			sm.machineState = standby
			sm.timeStateEntered = time.Now()
		}
	case shutdown2:
		sm.RelayShutdown = true
		sm.RelayInjector = false
		secondsSince := time.Since(sm.timeStateEntered).Seconds()
		if secondsSince >= 10 {
			sm.RelayShutdown = false
			sm.machineState = shutdownMonitor2
			sm.timeStateEntered = time.Now()
		}
	case shutdownMonitor2:
		if sm.state.GpioDigitalWaterOn { // if water is still on
			// TODO display "irrigation system failed to shutdown"
			// TODO wait for user acknowledge
		} else {
			// TODO wait for user acknowledge
			sm.Armed = false
			sm.machineState = standby
			sm.timeStateEntered = time.Now()
		}

	}

}
