package iscontrol

import "github.com/simpleiot/simpleiot/farmation/isdata"

// StateMachine ..
type StateMachine struct {
	config *isdata.Config
	state  *isdata.State

	// state machine internals
	State State

	// state machine outputs
	RelayShutdown bool
}

// State of machine
type State int

// define valid states
const ()

// NewStateMachine creates a new state machine
func NewStateMachine(config *isdata.Config, state *isdata.State) *StateMachine {
	return &StateMachine{
		config: config,
		state:  state,
	}
}

// Run executes state machine
func (sm *StateMachine) Run() {

}
