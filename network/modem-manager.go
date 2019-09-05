package network

import (
	"log"
	"time"
)

// State is used to describe the state of a device
type State int

// define valid states
const (
	StateNotDetected State = iota
	StateConfigure
	StateDisconnected
	StateConnected
	StateError
)

func (s State) String() string {
	switch s {
	case StateNotDetected:
		return "Not detected"
	case StateConfigure:
		return "Configuring device"
	case StateDisconnected:
		return "Device disconected"
	case StateConnected:
		return "Device connected"
	default:
		return "unknown"
	}
}

// ModemManager is used to configure a modem and manage the modem
// lifecycle.
type ModemManager struct {
	modem      *Modem
	state      State
	stateStart time.Time
}

// NewModemManager constructor
func NewModemManager(modem *Modem) *ModemManager {
	return &ModemManager{
		modem: modem,
	}
}

func (mm *ModemManager) setState(state State) {
	if state != mm.state {
		log.Printf("Modem state: %v -> %v", mm.state, state)
		mm.state = state
		mm.stateStart = time.Now()
	}
}

func (mm *ModemManager) stateMachine(modemState ModemState) {
	count := 0

	if !modemState.Detected {
		mm.setState(StateNotDetected)
	}

	for {
		count++
		if count > 10 {
			log.Println("modem state machine ran too many times")
			return
		}
		switch mm.state {
		case StateNotDetected:
			if modemState.Detected {
				mm.setState(StateConfigure)
				continue
			}
			// TODO add timeout and toggle hardware reset
		case StateConfigure:
			if time.Since(mm.stateStart) > time.Minute*2 {
				log.Println("giving up configuring modem")
				mm.setState(StateError)
			}
			err := mm.modem.Configure()
			if err != nil {
				return
			}
			mm.setState(StateDisconnected)
			continue
		case StateDisconnected:
			if modemState.Connected {
				mm.setState(StateConnected)
			}

			if time.Since(mm.stateStart) > time.Minute {
				log.Println("Modem disconnected timeout -- resetting modem")
				err := mm.modem.Reset()
				if err != nil {
					log.Println("Error resetting modem: ", err)
				}
				mm.setState(StateNotDetected)
			}
		case StateConnected:
			if !modemState.Connected {
				mm.setState(StateDisconnected)
			}
		}

		// if we want to re-run state machine, must execute continue above
		break
	}
}

// GetState must be called periodically to process the modem life cycle
// -- perhaps every 10s
func (mm *ModemManager) GetState() (ModemState, error) {
	s, err := mm.modem.GetState()
	if err != nil {
		return s, err
	}

	mm.stateMachine(s)

	return s, nil
}
