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
	machineState            StateMachineState
	timeStateEntered        time.Time
	lastGoodFlow            time.Time
	lastGoodPressure        time.Time
	lastPresDialogDisplayed time.Time
	waitingWaterDisplayed   bool
	waitingIrrDisplayed     bool
	//tankAlertDisplayed      bool

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

// StateMachineState is the state of the machine
type StateMachineState int

// define valid states
const (
	// states for monitor only
	monitorOnly StateMachineState = iota

	// states for monitor/shutdown
	// the Start/End markers are not actually used
	// but allow us to easily detect this group of states
	monitorShutdownStart
	standby
	monitoringFlow
	shutdownStart
	shutdown1
	shutdownMonitor1
	shutdown2        // UNUSED
	shutdownMonitor2 // UNUSED
	disarm
	shutdownDialog
	shutdownDialogAck
	notifiedSoDisarm
	monitorShutdownEnd

	// states for monitor/batch
)

func (s StateMachineState) String() string {
	switch s {
	case monitorOnly:
		return "monitorOnly"
	case standby:
		return "standby"
	case monitoringFlow:
		return "monitoringFlow"
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
func NewStateMachine(config *isdata.Config, state *isdata.State, initialState StateMachineState) *StateMachine {

	log.Println("Initial State:", initialState)
	return &StateMachine{
		machineState:            initialState,
		config:                  config,
		state:                   state,
		timeStateEntered:        time.Now(),
		lastGoodFlow:            time.Now(),
		lastGoodPressure:        time.Now(),
		lastPresDialogDisplayed: time.Now(),
	}
}

// SetState is used in iscontrol.Run() to set the machine state to
// a stored value from the system state on startup, and is also used
// throughout the state machine
func (sm *StateMachine) SetState(newState StateMachineState) isdata.UpdateStateMachineState {
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

	return isdata.UpdateStateMachineState(sm.machineState)
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

	// Define the key at which to access the state machine dialog
	smKey := "StateMachine"

	// before running state machine:
	if sm.inMonitorShutdownStates() {
		if sm.config.OperatingMode != isdata.ISOperatingModeMonitorAndShutdown &&
			sm.config.OperatingMode != isdata.ISOperatingModeMonitorAndNotify {

			return append(ret, sm.SetState(monitorOnly))
		}
		// If disarmed in non-shutdown and non-standbyWaiting states, go to standby
		// Is this only for if the user disarms in monitoringFlow state?
		if !sm.inShutdownStates() &&
			!sm.config.Arm {
			ret = append(ret, sm.SetState(standby))
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

		if sm.config.OperatingMode == isdata.ISOperatingModeMonitorAndShutdown ||
			sm.config.OperatingMode == isdata.ISOperatingModeMonitorAndNotify {
			if !sm.config.Arm {
				ret = append(ret, sm.SetState(standby))
			} else { // if armed on startup, go to monitoring flow
				ret = append(ret, sm.SetState(monitoringFlow))
			}
		}

		/*if int(sm.state.CurrentTankVolume) > sm.config.TankAlertVolume {
			sm.tankAlertDisplayed = false
		}

		if sm.config.TankAlertOn &&
			int(sm.state.CurrentTankVolume) <= sm.config.TankAlertVolume &&
			!sm.tankAlertDisplayed &&
			!sm.state.DialogStateMachine.Active {
			sm.tankAlertDisplayed = true
			return append(ret, isdata.UpdateDialogStateMachineMessage("Tank volume below\nalert level"))
		}*/

	// below states are for Monitor and Shutdown and Monitor and Notify modes
	case standby:
		controlInjector()

		sm.CurrentLedState = LedGreenBlnk

		if sm.config.Arm {
			ret = append(ret, sm.SetState(monitoringFlow))
		}

		/*if int(sm.state.CurrentTankVolume) > sm.config.TankAlertVolume {
			sm.tankAlertDisplayed = false
		}

		if sm.config.TankAlertOn &&
			int(sm.state.CurrentTankVolume) <= sm.config.TankAlertVolume &&
			!sm.tankAlertDisplayed &&
			!sm.state.DialogStateMachine.Active {
			sm.tankAlertDisplayed = true
			return append(ret, isdata.UpdateDialogStateMachineMessage("Tank volume below\nalert level"))
		}*/

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

		/*if int(sm.state.CurrentTankVolume) > sm.config.TankAlertVolume {
			sm.tankAlertDisplayed = false
		}*/

		// Display dialogs
		waterMsg := "Waiting for water"
		irrMsg := "Waiting for irrigator"
		lowPresMsg := "Pressure below\nshutdown threshold"
		//lowTankMsg := "Tank volume below\nalert level"

		if sm.state.InputWaterOn == isdata.InputStateOff &&
			!sm.waitingWaterDisplayed &&
			!sm.state.Dialogs[smKey].Active {
			sm.waitingWaterDisplayed = true
			return append(ret, isdata.UpdateDialogStateMachine{"Notice", waterMsg})
		}

		if sm.state.InputIrrigator == isdata.InputStateOff &&
			!sm.waitingIrrDisplayed &&
			!sm.state.Dialogs[smKey].Active {
			sm.waitingIrrDisplayed = true
			return append(ret, isdata.UpdateDialogStateMachine{"Notice", irrMsg})
		}

		if sm.config.PressureShutdownEnabled &&
			sm.RelayInjector &&
			lowPressure &&
			time.Since(sm.lastGoodPressure) >= time.Duration(5)*time.Second &&
			time.Since(sm.lastPresDialogDisplayed) >= time.Duration(30)*time.Second {
			sm.lastPresDialogDisplayed = time.Now()
			return append(ret, isdata.UpdateDialogStateMachine{"Warning", lowPresMsg})
		}

		/*if sm.config.TankAlertOn &&
			int(sm.state.CurrentTankVolume) <= sm.config.TankAlertVolume &&
			!sm.tankAlertDisplayed &&
			!sm.state.DialogStateMachine.Active {
			sm.tankAlertDisplayed = true
			return append(ret, isdata.UpdateDialogStateMachineMessage(lowTankMsg))
		}*/

		// Close dialogs if problem goes away
		if sm.state.InputWaterOn != isdata.InputStateOff &&
			sm.state.Dialogs[smKey].Active &&
			sm.state.Dialogs[smKey].Message == waterMsg {
			return append(ret, isdata.DialogClose{smKey})
		}

		if sm.state.InputIrrigator != isdata.InputStateOff &&
			sm.state.Dialogs[smKey].Active &&
			sm.state.Dialogs[smKey].Message == irrMsg {
			return append(ret, isdata.DialogClose{smKey})
		}

		if !lowPressure &&
			sm.state.Dialogs[smKey].Active &&
			sm.state.Dialogs[smKey].Message == lowPresMsg {
			return append(ret, isdata.DialogClose{smKey})
		}

		// ***This situation will never happen***
		/*if int(sm.state.CurrentTankVolume) > sm.config.TankAlertVolume &&
			sm.state.DialogStateMachine.Active &&
			sm.state.DialogStateMachine.Message == lowTankMsg {
			return isdata.UpdateDialogStateMachineClose{}
		}*/

		alarmRecognizeDuration := time.Duration(sm.config.AlarmRecognizeSec) * time.Second

		// The following switch statement is used only to determine next case.
		// Keep all other logic above.

		msg := "Alarm state entered: flow\noff target."

		switch {
		case sm.state.FlowStatus == isdata.FlowStatusOffTarget &&
			sm.RelayInjector &&
			time.Since(sm.lastGoodFlow) >= alarmRecognizeDuration:

			ret = append(ret, sm.SetState(shutdown1))
			ret = append(ret, data.Sample{
				Type:  isdata.SampleTypeFaultFlowOff,
				Time:  time.Now(),
				Value: sm.state.FlowRate,
				Attributes: map[string]float64{
					"inputInjector":  float64(sm.state.InputInjector),
					"inputWaterOn":   float64(sm.state.InputWaterOn),
					"inputIrrigator": float64(sm.state.InputIrrigator),
				},
			})

			return append(ret, isdata.UpdateDialogStateMachine{"Notice", msg})

		case sm.config.PressureShutdownEnabled &&
			lowPressure &&
			sm.RelayInjector &&
			time.Since(sm.lastGoodPressure) >= alarmRecognizeDuration:

			ret = append(ret, sm.SetState(shutdown1))

			// if flow is off target as well, prioritize this fault
			if sm.state.FlowStatus == isdata.FlowStatusOffTarget &&
				time.Since(sm.lastGoodFlow) >= alarmRecognizeDuration/3 {
				ret = append(ret, data.Sample{
					Type:  isdata.SampleTypeFaultFlowOff,
					Time:  time.Now(),
					Value: sm.state.FlowRate,
					Attributes: map[string]float64{
						"inputInjector":  float64(sm.state.InputInjector),
						"inputWaterOn":   float64(sm.state.InputWaterOn),
						"inputIrrigator": float64(sm.state.InputIrrigator),
					},
				})

				return append(ret, isdata.UpdateDialogStateMachine{"Notice", msg})
			}
			ret = append(ret, data.Sample{
				Type:  isdata.SampleTypeFaultPresLow,
				Time:  time.Now(),
				Value: sm.state.PressureMin,
				Attributes: map[string]float64{
					"inputInjector":     float64(sm.state.InputInjector),
					"inputWaterOn":      float64(sm.state.InputWaterOn),
					"inputIrrigator":    float64(sm.state.InputIrrigator),
					"shutdownThreshold": sm.config.PressureShutdownLow,
				},
			})
			msg = "Alarm state entered: low\npressure reading of " +
				strconv.FormatFloat(sm.state.PressureMin, 'f', 0, 64)

			return append(ret, isdata.UpdateDialogStateMachine{"Notice", msg})

		case sm.state.PressureMax >= float64(sm.config.HighPres):

			ret = append(ret, sm.SetState(shutdown1))
			ret = append(ret, data.Sample{
				Type:  isdata.SampleTypeFaultPresHigh,
				Time:  time.Now(),
				Value: sm.state.PressureMax,
				Attributes: map[string]float64{
					"inputInjector":     float64(sm.state.InputInjector),
					"inputWaterOn":      float64(sm.state.InputWaterOn),
					"inputIrrigator":    float64(sm.state.InputIrrigator),
					"shutdownThresHigh": float64(sm.config.HighPres),
				},
			})

			msg = "Alarm state entered: high\npressure reading of " +
				strconv.FormatFloat(sm.state.PressureMax, 'f', 0, 64)

			return append(ret, isdata.UpdateDialogStateMachine{"Notice", msg})
		}

	case shutdown1:

		sm.RelayShutdown = true
		sm.CurrentLedState = LedRed

		dlgStateMachine := *sm.state.Dialogs["StateMachine"]

		// if the user has acknowledged the dialog, disarm
		if !dlgStateMachine.Active {
			if dlgStateMachine.Ack {
				ret = append(ret, sm.SetState(disarm))
			} else { // if the system was restarted and we are still in this state, refire the dialog
				return append(ret, isdata.UpdateDialogStateMachine{"Notice", "Alarm state"})

			}
		}

		// If user toggles the arm switch, alarm state is exited
		if !sm.config.Arm {
			ret = append(ret, sm.SetState(standby))
			return append(ret, isdata.UpdateDialogStateMachine{"Notice", "User disarmed" +
				" system.\nAlarm state exited."})
		}

	/*case shutdownMonitor1:

	sm.CurrentLedState = LedRed

	if sm.elapsed() > 10*time.Second {
		sm.SetState(disarm)
	}

	// If user toggles the arm switch, alarm state is exited
	if !sm.config.Arm {
		sm.SetState(standby)
		return append(ret, isdata.UpdateDialogStateMachine{"Notice", "User disarmed system.\nShutdown aborted."})
	}*/

	case disarm:
		sm.CurrentLedState = LedRed

		if sm.config.Arm {
			return append(ret, isdata.UpdateDisarm{})
		}
		ret = append(ret, sm.SetState(standby))

		/*
			case shutdownDialog:

					sm.CurrentLedState = LedRed

					sm.SetState(shutdownDialogAck)

					if sm.state.InputWaterOn == isdata.InputStateOn {
						return append(ret, isdata.UpdateDialogStateMachine{"Notice", "Failed to shutdown irrigator"}, data.Sample{
							Type: isdata.SampleTypeFaultShutdown,
							Time: time.Now(),
							Attributes: map[string]float64{
								"inputInjector":  float64(sm.state.InputInjector),
								"inputWaterOn":   float64(sm.state.InputWaterOn),
								"inputIrrigator": float64(sm.state.InputIrrigator),
							},
						})
					}

					return append(ret, isdata.UpdateDialogStateMachine{"Notice", "System shut down irrigator"})

				case shutdownDialogAck:

					sm.CurrentLedState = LedRed

					if !sm.state.Dialogs[smKey].Active {
						sm.SetState(standby)
					}
		*/
	}

	return ret
}
