package iscontrol

import (
	"log"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StateMachine ..
type StateMachine struct {
	// state machine inputs
	config *isdata.Config
	state  *isdata.State

	// state machine internals
	machineState            state
	timeStateEntered        time.Time
	lastGoodFlow            time.Time
	lastGoodPressure        time.Time
	lastPresDialogDisplayed time.Time
	waitingWaterDisplayed   bool
	waitingIrrDisplayed     bool
	tankAlertDisplayed      bool

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
	case disarm:
		return "disarm"
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
	default:
		return strconv.Itoa(int(s))
	}
}

// NewStateMachine creates a new state machine
func NewStateMachine(config *isdata.Config, state *isdata.State) *StateMachine {
	return &StateMachine{
		config:                  config,
		state:                   state,
		timeStateEntered:        time.Now(),
		lastGoodFlow:            time.Now(),
		lastGoodPressure:        time.Now(),
		lastPresDialogDisplayed: time.Now(),
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
func (sm *StateMachine) Run() (ret []interface{}) {

	// set relays to false and set to other values as needed in case statements
	sm.RelayShutdown = false
	sm.RelayInjector = false

	// before running state machine:
	if sm.inMonitorShutdownStates() {
		if sm.config.OperatingMode != isdata.ISOperatingModeMonitorAndShutdown {
			sm.setState(monitorOnly)
			return
		}
		// if disarmed in non-shutdown and non-standbyWaiting states, go to standby
		if !sm.inShutdownStates() &&
			!sm.config.Arm {
			sm.setState(standby)
		}
	}

	controlInjector := func() {
		sm.RelayInjector = sm.state.InputInjector == isdata.InputStateOn &&
			(sm.state.InputWaterOn == isdata.InputStateNA || sm.state.InputWaterOn == isdata.InputStateOn) &&
			(sm.state.InputIrrigator == isdata.InputStateNA || sm.state.InputIrrigator == isdata.InputStateOn)
	}

	switch sm.machineState {
	case monitorOnly:
		controlInjector()

		sm.CurrentLedState = LedGreenBlnk

		if sm.config.OperatingMode == isdata.ISOperatingModeMonitorAndShutdown {
			if !sm.config.Arm {
				sm.setState(standby)
			} else { // if armed on startup, go to monitoring flow
				sm.setState(monitoringFlow)
			}
		}

		if int(sm.state.CurrentTankVolume) > sm.config.TankAlertVolume {
			sm.tankAlertDisplayed = false
		}

		if sm.config.TankAlertOn &&
			int(sm.state.CurrentTankVolume) <= sm.config.TankAlertVolume &&
			!sm.tankAlertDisplayed &&
			!sm.state.DialogStateMachine.Active {
			sm.tankAlertDisplayed = true
			return append(ret, isdata.UpdateDialogStateMachineMessage("Tank volume below\nalert level"))
		}

	// below states are for monitor/shutdown
	case standby:
		controlInjector()

		sm.CurrentLedState = LedGreenBlnk

		if sm.config.Arm {
			sm.setState(monitoringFlow)
		}

		if int(sm.state.CurrentTankVolume) > sm.config.TankAlertVolume {
			sm.tankAlertDisplayed = false
		}

		if sm.config.TankAlertOn &&
			int(sm.state.CurrentTankVolume) <= sm.config.TankAlertVolume &&
			!sm.tankAlertDisplayed &&
			!sm.state.DialogStateMachine.Active {
			sm.tankAlertDisplayed = true
			return append(ret, isdata.UpdateDialogStateMachineMessage("Tank volume below\nalert level"))
		}

	case monitoringFlow:
		controlInjector()

		lowPressure := sm.state.PressureMin < sm.config.PressureShutdownLow

		if sm.RelayInjector &&
			(sm.state.FlowStatus == isdata.FlowStatusOffTarget ||
				(sm.config.PressureShutdownEnabled && lowPressure)) {
			sm.CurrentLedState = LedRedBlnk
		} else {
			sm.CurrentLedState = LedGreen
		}

		// Reset time stamps and dialogs displayed booleans
		// for flow and pressure timestamps, if parameter ok OR pump not
		// on, reset timestamp
		if !(sm.state.FlowStatus == isdata.FlowStatusOffTarget) ||
			!sm.RelayInjector {
			sm.lastGoodFlow = time.Now()
		}

		if !lowPressure ||
			!sm.RelayInjector {
			sm.lastGoodPressure = time.Now()
		}

		if sm.state.InputWaterOn != isdata.InputStateOff {
			sm.waitingWaterDisplayed = false
		}

		if sm.state.InputIrrigator != isdata.InputStateOff {
			sm.waitingIrrDisplayed = false
		}

		if int(sm.state.CurrentTankVolume) > sm.config.TankAlertVolume {
			sm.tankAlertDisplayed = false
		}

		// Display dialogs
		waterMsg := "Waiting for water"
		irrMsg := "Waiting for irrigator"
		lowPresMsg := "Pressure below\nshutdown threshold"
		lowTankMsg := "Tank volume below\nalert level"

		if sm.state.InputWaterOn == isdata.InputStateOff &&
			!sm.waitingWaterDisplayed &&
			!sm.state.DialogStateMachine.Active {
			sm.waitingWaterDisplayed = true
			return append(ret, isdata.UpdateDialogStateMachineMessage(waterMsg))
		}

		if sm.state.InputIrrigator == isdata.InputStateOff &&
			!sm.waitingIrrDisplayed &&
			!sm.state.DialogStateMachine.Active {
			sm.waitingIrrDisplayed = true
			return append(ret, isdata.UpdateDialogStateMachineMessage(irrMsg))
		}

		if sm.config.PressureShutdownEnabled &&
			sm.RelayInjector &&
			lowPressure &&
			time.Since(sm.lastGoodPressure) >= time.Duration(5)*time.Second &&
			time.Since(sm.lastPresDialogDisplayed) >= time.Duration(30)*time.Second {
			sm.lastPresDialogDisplayed = time.Now()
			return append(ret, isdata.UpdateDialogStateMachineMessage(lowPresMsg))
		}

		if sm.config.TankAlertOn &&
			int(sm.state.CurrentTankVolume) <= sm.config.TankAlertVolume &&
			!sm.tankAlertDisplayed &&
			!sm.state.DialogStateMachine.Active {
			sm.tankAlertDisplayed = true
			return append(ret, isdata.UpdateDialogStateMachineMessage(lowTankMsg))
		}

		// Close dialogs if problem goes away
		if sm.state.InputWaterOn != isdata.InputStateOff &&
			sm.state.DialogStateMachine.Active &&
			sm.state.DialogStateMachine.Message == waterMsg {
			return append(ret, isdata.UpdateDialogStateMachineClose{})
		}

		if sm.state.InputIrrigator != isdata.InputStateOff &&
			sm.state.DialogStateMachine.Active &&
			sm.state.DialogStateMachine.Message == irrMsg {
			return append(ret, isdata.UpdateDialogStateMachineClose{})
		}

		if !lowPressure &&
			sm.state.DialogStateMachine.Active &&
			sm.state.DialogStateMachine.Message == lowPresMsg {
			return append(ret, isdata.UpdateDialogStateMachineClose{})
		}

		// ***This situation will never happen***
		/*if int(sm.state.CurrentTankVolume) > sm.config.TankAlertVolume &&
			sm.state.DialogStateMachine.Active &&
			sm.state.DialogStateMachine.Message == lowTankMsg {
			return isdata.UpdateDialogStateMachineClose{}
		}*/

		alarmRecognizeDuration := time.Duration(sm.config.AlarmRecognizeSec) * time.Second

		// the following switch statement is used only to determine next case. Keep all other logic
		// above.
		switch {
		case sm.state.FlowStatus == isdata.FlowStatusOffTarget &&
			sm.RelayInjector &&
			time.Since(sm.lastGoodFlow) >= alarmRecognizeDuration:
			sm.setState(disarm)
			return append(ret, data.Sample{
				Type:    isdata.SampleTypeFault,
				SubType: isdata.SampleSubTypeFaultFlow,
				Time:    time.Now(),
				Value:   sm.state.FlowRate,
			})

		case sm.config.PressureShutdownEnabled &&
			lowPressure &&
			sm.RelayInjector &&
			time.Since(sm.lastGoodPressure) >= alarmRecognizeDuration:
			sm.setState(disarm)

			// if flow is off target as well, prioritize this fault
			if sm.state.FlowStatus == isdata.FlowStatusOffTarget &&
				time.Since(sm.lastGoodFlow) >= alarmRecognizeDuration/3 {
				return append(ret, data.Sample{
					Type:    isdata.SampleTypeFault,
					SubType: isdata.SampleSubTypeFaultFlow,
					Time:    time.Now(),
					Value:   sm.state.FlowRate,
				})
			}
			return append(ret, data.Sample{
				Type:    isdata.SampleTypeFault,
				SubType: isdata.SampleSubTypeFaultPres,
				Time:    time.Now(),
				Value:   sm.state.PressureMin,
			})
		}

	case disarm:

		sm.CurrentLedState = LedRed

		if sm.config.Arm {
			return append(ret, isdata.UpdateDisarm{})
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
				return append(ret, isdata.UpdateDialogStateMachineMessage("failed to shutdown"), data.Sample{
					Type:    isdata.SampleTypeFault,
					SubType: isdata.SampleSubTypeFaultShutdown,
					Time:    time.Now(),
				})
			}

			return append(ret, isdata.UpdateDialogStateMachineMessage("system shut down"))
		}
	}

	return
}
