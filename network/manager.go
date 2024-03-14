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
	StateConfigure
	StateConnecting
	StateConnected
	StateError
)

func (s State) String() string {
	switch s {
	case StateNotDetected:
		return "Not detected"
	case StateConfigure:
		return "Configure"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateError:
		return "Error"
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

// nextInterface resets state and checks if there is another interface to try.
// If there are no more interfaces, sets state to error.
func (m *Manager) nextInterface() {
	m.interfaceIndex++
	if m.interfaceIndex >= len(m.interfaces) {
		m.interfaceIndex = 0
		log.Println("Network: no more interfaces, go to error state")
		m.setState(StateError)
		return
	}

	m.setState(StateNotDetected)

	log.Println("Network: trying next interface:", m.Desc())
}

func (m *Manager) connect() error {
	if len(m.interfaces) <= 0 {
		return errors.New("No interfaces to connect to")
	}

	return m.interfaces[m.interfaceIndex].Connect()
}

func (m *Manager) configure() (InterfaceConfig, error) {
	if len(m.interfaces) <= 0 {
		return InterfaceConfig{}, errors.New("No interfaces to configure")
	}

	return m.interfaces[m.interfaceIndex].Configure()
}

// Reset resets all network interfaces
func (m *Manager) Reset() {
	for _, i := range m.interfaces {
		err := i.Reset()
		if err != nil {
			log.Println("Error resetting interface:", err)
		}
	}
}

// Run must be called periodically to process the network life cycle
// -- perhaps every 10s
func (m *Manager) Run() (State, InterfaceConfig, InterfaceStatus) {
	count := 0

	status := InterfaceStatus{}
	config := InterfaceConfig{}

	// state machine for network manager
	for {
		count++
		if count > 10 {
			log.Println("network state machine ran too many times")
			if time.Since(m.stateStart) > time.Second*15 {
				log.Println("Network: timeout:", m.Desc())
				m.nextInterface()
			}

			return m.state, config, status
		}

		if m.state != StateError {
			var err error
			status, err = m.getStatus()
			if err != nil {
				log.Println("Error getting interface status:", err)
				continue
			}
		}

		switch m.state {
		case StateNotDetected:
			// give ourselves 15 seconds or so in detecting state
			// in case we just reset the devices
			if status.Detected {
				log.Printf("Network: %v detected\n", m.Desc())
				m.setState(StateConfigure)
				continue
			} else if time.Since(m.stateStart) > time.Second*15 {
				log.Println("Network: timeout detecting:", m.Desc())
				m.nextInterface()
				continue
			}

		case StateConfigure:
			try := 0
			for ; try < 3; try++ {
				if try > 0 {
					log.Println("Trying again ...")
				}
				var err error
				config, err = m.configure()
				if err != nil {
					log.Printf("Error configuring device: %v: %v\n",
						m.Desc(), err)

					continue
				}

				break
			}

			if try < 3 {
				m.setState(StateConnecting)
			} else {
				log.Println("giving up configuring device:", m.Desc())
				m.nextInterface()
			}

		case StateConnecting:
			if status.Connected {
				log.Printf("Network: %v connected\n", m.Desc())
				m.setState(StateConnected)
			} else {
				if time.Since(m.stateStart) > time.Minute {
					log.Println("Network: timeout connecting:", m.Desc())
					m.nextInterface()
					continue
				}

				// try again to connect
				err := m.connect()
				if err != nil {
					log.Println("Error connecting:", err)
				}
			}

		case StateConnected:
			if !status.Connected {
				// try to reconnect
				m.setState(StateConnecting)
			}

		case StateError:
			if time.Since(m.stateStart) > time.Minute {
				log.Println("Network: reset and try again ...")
				m.Reset()
				m.setState(StateNotDetected)
			}
		}

		// if we want to re-run state machine, must execute continue above
		break
	}

	return m.state, config, status
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
