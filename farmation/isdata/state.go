package isdata

import (
	"runtime"
	"time"

	"github.com/blang/semver"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/version"
	"github.com/simpleiot/simpleiot/network"
)

// State contains the current injectory sentry state.
// Note, these type is stored directly in database,
// so you can't ever change types of fields, or remove values from
// consts. It is safest to never delete fields, but rather comment
// that they are no longer used, so they don't accidently get
// reused in the future with a different type.
type State struct {
	SystemType SystemType `json:"systemType"`

	// FlowRate defines the current flow rate of the system in GPH
	FlowRate float64 `json:"flowRate"`
	// AvgFlowRate defines the average flow rate of the system since it
	// was last armed
	AvgFlowRate       float64      `json:"avgFlowRate"`
	AvgFlowRateStart  time.Time    `json:"avgFlowRateStart"`
	FlowRateMin       float64      `json:"flowRateMin"`
	FlowRateMax       float64      `json:"flowRateMax"`
	BatchApplied      float64      `json:"batchApplied"`
	BatchRemaining    float64      `json:"batchRemaining"`
	Total1            float64      `json:"total1"`
	Total2            float64      `json:"total2"`
	LifetimeTotal     float64      `json:"lifetimeTotal"`
	FlowPulseCount    int          `json:"flowPulseCount"`
	CurrentTankVolume float64      `json:"currentTankVolume"`
	NetworkState      NetworkState `json:"networkState"`

	FieldStates    [][5]ProductState `json:"fieldStates"`
	GpsPos         GpsPos            `json:"gpsPos"`
	FlowStatus     FlowStatus        `json:"flowStatus"`
	PressureMin    float64           `json:"pressureMin"`
	PressureMax    float64           `json:"pressureMax"`
	PressureAvg    float64           `json:"pressureAvg"`
	PressureVRef   float64           `json:"pressureVRef"`
	PressureVSense float64           `json:"pressureVSense"`

	// Gpio's
	GpioDigitalInjector  bool `json:"gpioDigitalInjector"`
	GpioDigitalIrrigator bool `json:"gpioDigitalIrrigator"`
	GpioDigitalWaterOn   bool `json:"gpioDigitalWaterOn"`
	GpioDigitalIn        bool `json:"gpioDigitalIn"`

	GpioRelayInjectorEn bool `json:"gpioRelayInjectorEn"`
	GpioRelayShutdownEn bool `json:"gpioRelayShutdownEn"`
	GpioRelayAuxEn      bool `json:"gpioRelayAuxEn"`

	GpioRegValve1 bool
	GpioRegValve2 bool

	GpioStatusLedRed   bool `json:"gpioStatusLedRed"`
	GpioStatusLedGreen bool `json:"gpioStatusLedGreen"`

	// Virtual inputs based on panel type and pump control selection
	// the below are selected from Lindsay serial data, GPIOs, or set to off
	// based on user preferences
	InputWaterOn   InputState `json:"inputWaterOn"`
	InputIrrigator InputState `json:"inputIrrigator"`
	InputInjector  InputState `json:"inputInjector"`

	// Data from Lindsay panel
	LindsayRegs       LindsayStatusRegs `json:"lindsayRegs"`
	LindsayLastUpdate time.Time         `json:"lindsayLastUpdate"`

	// Faults
	FaultsActive FaultsActive `json:"faultsActive"`

	// Modal dialog describes a modal dialog message
	// OUTDATED: only for messages from state machine. Create new dialog
	// structs for other parts of the app.
	// NOTE: Add dialogs in method InitState()
	Dialogs map[string]*Dialog //map[string]Dialog

	OSVersion    semver.Version `json:"osVersion"`
	SerialNumber string         `json:"serialNumber"`

	// ViewMsg is set by the -msg flag on startup
	// determines if messages on the app channel are displayed in console
	ViewMsg bool `json:"viewMsg"`

	// Deleted Fields
	// PanelDefinition will be populated based on the panel detected
	// by the sense resistor.
	// PanelDefinition PanelDefinition `json:"panelConfig"`

	NetworkInterfaceConfig network.InterfaceConfig

	HWVersion int
}

// UpdateInputs update virtual inputs based on panel type and pump config
func (s *State) UpdateInputs(config *Config) {
	// set water on based on panel type
	switch config.PanelType {
	case PanelTypeStandardPump:
		s.InputWaterOn = BoolToInputState(s.GpioDigitalWaterOn)
		s.InputIrrigator = InputStateNA
	case PanelTypeStandardPivot:
		s.InputWaterOn = BoolToInputState(s.GpioDigitalWaterOn)
		s.InputIrrigator = BoolToInputState(s.GpioDigitalIrrigator)
	case PanelTypeLindsay:
		s.InputWaterOn = BoolToInputState(s.LindsayRegs.WaterOn())
		s.InputIrrigator = BoolToInputState(s.LindsayRegs.IrrigatorRunning())
	default:
		// for invalid/unsupported panels, simply default to GPIO inputs
		// or StandardPivot
		s.InputWaterOn = BoolToInputState(s.GpioDigitalWaterOn)
		s.InputIrrigator = BoolToInputState(s.GpioDigitalIrrigator)
	}

	switch config.UserPumpMode {
	case UserPumpModeOff:
		// force virtual inputs off to turn injector relay off
		s.InputInjector = InputStateOff
	case UserPumpModeOn:
		// force virtual inputs on to turn injector relay on
		s.InputInjector = InputStateOn
	case UserPumpModeInj:
		s.InputInjector = BoolToInputState(s.GpioDigitalInjector)
	case UserPumpModeAcc1:
		s.InputInjector = BoolToInputState(s.LindsayRegs.Accessory1On())
	case UserPumpModeAcc2:
		s.InputInjector = BoolToInputState(s.LindsayRegs.Accessory2On())
	default:
		s.InputInjector = InputStateOff
	}
}

// InputState is a type that describes if the input is avaliable, and what its state is
type InputState int

// define value input states
const (
	InputStateNA InputState = iota
	InputStateOff
	InputStateOn
)

func (is InputState) String() string {
	switch is {
	case InputStateNA:
		return "NA"
	case InputStateOff:
		return "off"
	case InputStateOn:
		return "on"
	}

	return "unknown"
}

// BoolToInputState converts a bool to input state, assuming
// it is not InputStateNA
func BoolToInputState(v bool) InputState {
	if v {
		return InputStateOn
	}

	return InputStateOff
}

// FaultsActive defines a slice for FaultsActive
type FaultsActive []data.Sample

// ActiveFaults returns true if any fault is active and false otherwise
func (fa FaultsActive) ActiveFaults() bool {
	if len(fa) >= 1 {
		return true
	}
	return false
}

// Dialog defines a modal dialog that must be acknowledged
type Dialog struct {
	Priority int
	Active   bool
	Heading  string
	Message  string
}

// Define dialog priority ranking scale: 0 is highest priority
const (
	DialogPriorityReboot int = iota
	DialogPriorityRestart
	DialogPriorityUpdate
	DialogPriorityPanelDetect
	DialogPriorityUnknownVisionState
	DialogPriorityApp
	DialogPriorityArm
	DialogPriorityArmReq
	DialogPriorityStateMachine
	DialogPriorityExport
)

// DialogHighestPriority returns the key to the highest priority active
// dialog in the Dialogs map
func (s State) DialogHighestPriority() (key string) {
	for k, dlg := range s.Dialogs {
		// lower value of Priority field means higher priority
		if dlg.Active &&
			(key == "" || dlg.Priority < s.Dialogs[key].Priority) {
			key = k
		}
	}
	return key
}

// SystemType describes the system type
type SystemType int

// define valid system types
const (
	SystemTypeIS SystemType = iota
	SystemTypeISSim
)

// FlowStatus describes the overall system of flow control
type FlowStatus int

// possible system status values
const (
	FlowStatusArmedOk FlowStatus = iota
	FlowStatusOffTarget
)

// ProductState defines the state of a product
type ProductState struct {
	Total float64
}

// GpsPos represents a GPS position
type GpsPos struct {
	Time    time.Time
	Lat     float64
	Long    float64
	Fix     int
	NumSats int
}

// InitState initializes multiple states
func InitState(s *State) (dirty bool) {
	for len(s.FieldStates) < 30 {
		s.FieldStates = append(s.FieldStates, [5]ProductState{})
		dirty = true
	}

	if runtime.GOARCH == "arm" {
		s.SystemType = SystemTypeIS
	} else {
		s.SystemType = SystemTypeISSim
	}

	s.FlowRate = 0

	s.PressureMin = 0
	s.PressureAvg = 0
	s.PressureMax = 0
	s.PressureVRef = 0
	s.PressureVSense = 0

	s.GpioDigitalInjector = false
	s.GpioDigitalIrrigator = false
	s.GpioDigitalWaterOn = false
	s.GpioDigitalIn = false

	s.GpioRelayInjectorEn = false
	s.GpioRelayShutdownEn = false
	s.GpioRelayAuxEn = false

	s.GpioRegValve1 = false
	s.GpioRegValve2 = false

	s.GpioStatusLedRed = false
	s.GpioStatusLedGreen = false

	// Initialize all necessary dialogs
	// Static messages and headings are initialized here,
	// but text that has variable content is set elsewhere
	s.Dialogs = make(map[string]*Dialog)
	s.Dialogs["Reboot"] = &Dialog{
		Priority: DialogPriorityReboot,
		Heading:  "Notice",
		Message:  "Reboot started, please wait",
	}
	s.Dialogs["Restart"] = &Dialog{
		Priority: DialogPriorityRestart,
		Heading:  "Notice",
		Message: "The timezone was changed,\nso the Injector " +
			"Sentry will\nbe restarted.",
	}
	s.Dialogs["Update"] = &Dialog{
		Priority: DialogPriorityUpdate,
		Heading:  "Notice",
	}
	s.Dialogs["Arm"] = &Dialog{
		Priority: DialogPriorityArm,
		Heading:  "Error",
	}
	s.Dialogs["UnknownVisionState"] = &Dialog{
		Priority: DialogPriorityUnknownVisionState,
		Heading:  "Warning",
	}
	s.Dialogs["App"] = &Dialog{
		Priority: DialogPriorityApp,
		Heading:  "Warning",
	}
	s.Dialogs["Export"] = &Dialog{
		Priority: DialogPriorityExport,
	}
	s.Dialogs["StateMachine"] = &Dialog{
		Priority: DialogPriorityStateMachine,
	}
	s.Dialogs["ArmReq"] = &Dialog{
		Priority: DialogPriorityArmReq,
	}
	s.Dialogs["PanelDetect"] = &Dialog{
		Priority: DialogPriorityPanelDetect,
	}

	s.OSVersion, _ = version.ReadOSVersion()

	s.LindsayRegs = LindsayStatusRegs{}

	s.NetworkInterfaceConfig = network.InterfaceConfig{}

	return
}
