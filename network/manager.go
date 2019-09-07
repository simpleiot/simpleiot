package network

import (
	"errors"
	"log"
	"time"
)

// State is used to describe the network state
type State int

// define valid states
const (
	StateNotDetected State = iota
	StateConnecting
	StateConnected
	StateError
)

func (s State) String() string {
	switch s {
	case StateNotDetected:
		return "Not detected"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	default:
		return "unknown"
	}
}

// Manager is used to configure the network and manage the
// lifecycle.
type Manager struct {
	state          State
	stateStart     time.Time
	interfaces     []Interface
	errResetCnt    int
	errCnt         int
	interfaceIndex int
}

// NewManager constructor
func NewManager(errResetCnt int) *Manager {
	return &Manager{
		stateStart:  time.Now(),
		errResetCnt: errResetCnt,
	}
}

// AddInterface adds a network interface to the manager. Interfaces added first
// have higher priority
func (m *Manager) AddInterface(iface Interface) {
	m.interfaces = append(m.interfaces, iface)
}

func (m *Manager) setState(state State) {
	if state != m.state {
		log.Printf("Network state: %v -> %v", m.state, state)
		m.state = state
		m.stateStart = time.Now()
	}
}

func (m *Manager) getStatus() (InterfaceStatus, error) {
	if len(m.interfaces) <= 0 {
		return InterfaceStatus{}, nil
	}

	return m.interfaces[m.interfaceIndex].GetStatus()
}

// Desc returns current interface description
func (m *Manager) Desc() string {
	if len(m.interfaces) <= 0 {
		return "none"
	}

	return m.interfaces[m.interfaceIndex].Desc()
}

// nextInterface returns true if there is another interface to try, otherwise
// resets to zero and returns false
func (m *Manager) nextInterface() bool {
	m.interfaceIndex++
	if m.interfaceIndex >= len(m.interfaces) {
		m.interfaceIndex = 0
		log.Println("Network: no more interfaces to try")
		return false
	}

	log.Println("Network: trying next interface: ", m.Desc())
	return true
}

func (m *Manager) connect() error {
	if len(m.interfaces) <= 0 {
		return errors.New("No interfaces to connect to")
	}

	return m.interfaces[m.interfaceIndex].Connect()
}

// Reset resets all network interfaces
func (m *Manager) Reset() {
	for _, i := range m.interfaces {
		err := i.Reset()
		if err != nil {
			log.Println("Error resetting interface: ", err)
		}
	}
}

// Run must be called periodically to process the network life cycle
// -- perhaps every 10s
func (m *Manager) Run() (State, InterfaceStatus) {
	count := 0

	status := InterfaceStatus{}

	// state machine for network manager
	for {
		count++
		if count > 10 {
			log.Println("network state machine ran too many times")
			return m.state, status
		}

		var err error
		status, err = m.getStatus()
		if err != nil {
			log.Println("Error getting interface status: ", err)
			continue
		}

		switch m.state {
		case StateNotDetected:
			// give ourselves 15 seconds or so in detecting state
			// in case we just reset the devices
			if status.Detected {
				log.Printf("Network: %v detected\n", m.Desc())
				m.setState(StateConnecting)
				continue
			} else if time.Since(m.stateStart) > time.Second*15 {
				log.Println("Network: timeout detecting: ", m.Desc())
				if !m.nextInterface() {
					m.setState(StateError)
					break
				}

				continue
			}
		case StateConnecting:
			if status.Connected {
				log.Printf("Network: %v connected\n", m.Desc())
				m.setState(StateConnected)
			} else {
				if time.Since(m.stateStart) > time.Minute*5 {
					log.Println("Network: timeout connecting: ", m.Desc())
					if !m.nextInterface() {
						m.setState(StateError)
						break
					}

					continue
				}

				// try again to connect
				err := m.connect()
				if err != nil {
					log.Println("Error connecting: ", err)
				}
			}
		case StateConnected:
			if !status.Connected {
				// try to reconnect
				m.setState(StateConnecting)
			}
		case StateError:
			if time.Since(m.stateStart) > time.Minute {
				log.Println("Network: trying again ...")
				m.setState(StateNotDetected)
			}
		}

		// if we want to re-run state machine, must execute continue above
		break
	}

	return m.state, status
}

// Error is called any time there is a network error
// after errResetCnt errors are reached, we reset all the interfaces and
// start over
func (m *Manager) Error() {
	m.errCnt++

	if m.errCnt >= m.errResetCnt {
		m.Reset()
		m.setState(StateNotDetected)
		m.errCnt = 0
	}
}

// Success is called any time there is a network success
// so that we know to reset the internal error count
func (m *Manager) Success() {
	m.errCnt = 0
}
