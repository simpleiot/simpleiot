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
// Note, these types are stored directly in database,
// so you can't ever change types of fields, or remove values from
// consts. It is safest to never delete fields, but rather comment
// that they are no longer used, so they don't accidently get
// reused in the future with a different type.
type State struct {
	SystemType SystemType `json:"systemType"`

	DBConfig struct {
		DBVersion int `json:"dbVersion"`
	} `json:"dbConfig"`

	StateMachineState int `json:"stateMachineState"`

	// FlowRate defines the current flow rate of the system in GPH
	FlowRate              float64             `json:"flowRate"`
	TimeArmedAndInjOn     time.Time           `json:"timeArmedAndInjOn"`
	DurationArmedAndInjOn time.Duration       `json:"durationArmed"`
	FlowAverager          data.SampleAverager `json:"flowAverager"`
	FlowRateMin           float64             `json:"flowRateMin"`
	FlowRateMax           float64             `json:"flowRateMax"`
	BatchApplied          float64             `json:"batchApplied"`
	BatchRemaining        float64             `json:"batchRemaining"`
	Total1                float64             `json:"total1"`
	Total2                float64             `json:"total2"`
	LifetimeTotal         float64             `json:"lifetimeTotal"`
	FlowPulseCount        int                 `json:"flowPulseCount"`
	CurrentTankVolume     float64             `json:"currentTankVolume"`

	NetworkState NetworkState `json:"networkState"`

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
	GpioMainAuxPwr       bool `json:"gpioMainAuxPwr"`

	GpioRelayInjectorEn bool `json:"gpioRelayInjectorEn"`
	GpioRelayShutdownEn bool `json:"gpioRelayShutdownEn"`
	GpioRelayAuxEn      bool `json:"gpioRelayAuxEn"`

	GpioRegValve1 bool `json:"gpioRegValve1"`
	GpioRegValve2 bool `json:"gpioRegValve2"`

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
	Dialogs map[string]*Dialog `json:"dialogs"` //map[string]Dialog

	OSVersion    semver.Version `json:"osVersion"`
	SerialNumber string         `json:"serialNumber"`

	// ViewMsg is set by the -msg flag on startup
	// determines if messages on the app channel are displayed in console
	ViewMsg bool `json:"viewMsg"`

	// Deleted Fields
	// PanelDefinition will be populated based on the panel detected
	// by the sense resistor.
	// PanelDefinition PanelDefinition `json:"panelConfig"`

	NetworkInterfaceConfig network.InterfaceConfig `json:"networkInterfaceConfig"`

	HWVersion int `json:"hwVersion"`

	Location data.GpsPos `json:"location"`
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
	ID      int
	Active  bool
	Ack     bool
	Heading string
	Message string

	// Option to cancel the action the dialog is
	// warning about
	CancelActivated bool
}

// Define dialog ID's
// They are listed in order of priority; the lower the value,
// the higher the priority
// These constants can be reorderd, deleted, etc. because
// the dialogs are reinitialized in state at each app start
const (
	DialogShutdown int = iota
	DialogReboot
	DialogFactoryReset
	DialogSetTimezone
	DialogUpdate
	DialogPanelDetect
	DialogUnknownVisionState
	DialogApp
	DialogArm
	DialogArmReq
	DialogStateMachine
	DialogExport
	DialogResetTotalCurrent
	DialogResetTotal1
	DialogResetTotal2
)

// DialogHighestPriority returns the highest priority active
// dialog in the Dialogs map, as well as its key
func (s State) DialogHighestPriority() (highest *Dialog, key string) {
	for k, dlg := range s.Dialogs {
		// lower value of ID field means higher priority
		if dlg.Active &&
			(highest == nil || dlg.ID < highest.ID) {
			highest = dlg
			key = k
		}
	}
	return highest, key
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

	s.Location = data.GpsPos{}

	// Initialize all necessary dialogs
	// Static messages and headings are initialized here,
	// but text that has variable content is set elsewhere
	s.Dialogs = make(map[string]*Dialog)
	s.Dialogs["Shutdown"] = &Dialog{
		ID:      DialogShutdown,
		Heading: "Notice",
		Message: "Shutting down ...",
	}
	s.Dialogs["Reboot"] = &Dialog{
		ID:      DialogReboot,
		Heading: "Notice",
		Message: "Reboot started, please wait",
	}
	s.Dialogs["FactoryReset"] = &Dialog{
		ID:      DialogFactoryReset,
		Heading: "Warning",
		Message: "You are about to reset all configurable " +
			"values on this system to their default values from " +
			"the factory.",
		CancelActivated: true,
	}
	s.Dialogs["SetTimezone"] = &Dialog{
		ID:      DialogSetTimezone,
		Heading: "Notice",
		Message: "Application will now restart\nfor time " +
			"zone change to\ntake effect.",
		CancelActivated: true,
	}
	s.Dialogs["Update"] = &Dialog{
		ID:      DialogUpdate,
		Heading: "Notice",
	}
	s.Dialogs["PanelDetect"] = &Dialog{
		ID: DialogPanelDetect,
	}
	s.Dialogs["UnknownVisionState"] = &Dialog{
		ID:      DialogUnknownVisionState,
		Heading: "Warning",
	}
	s.Dialogs["App"] = &Dialog{
		ID:      DialogApp,
		Heading: "Warning",
	}
	s.Dialogs["Arm"] = &Dialog{
		ID:      DialogArm,
		Heading: "Error",
	}
	s.Dialogs["ArmReq"] = &Dialog{
		ID: DialogArmReq,
	}
	s.Dialogs["StateMachine"] = &Dialog{
		ID: DialogStateMachine,
	}
	s.Dialogs["Export"] = &Dialog{
		ID: DialogExport,
	}
	s.Dialogs["ResetTotalCurrent"] = &Dialog{
		ID:      DialogResetTotalCurrent,
		Heading: "Warning",
		Message: "You are about to reset the\ncurrent product " +
			"total to 0.",
		CancelActivated: true,
	}
	s.Dialogs["ResetTotal1"] = &Dialog{
		ID:              DialogResetTotal1,
		Heading:         "Warning",
		Message:         "You are about to reset\nTotal 1 to zero",
		CancelActivated: true,
	}
	s.Dialogs["ResetTotal2"] = &Dialog{
		ID:              DialogResetTotal2,
		Heading:         "Warning",
		Message:         "You are about to reset\nTotal 2 to zero",
		CancelActivated: true,
	}

	s.OSVersion, _ = version.ReadOSVersion()

	s.LindsayRegs = LindsayStatusRegs{}

	s.NetworkInterfaceConfig = network.InterfaceConfig{}

	if s.FlowAverager.SampleType == "" {
		s.FlowAverager = *data.NewSampleAverager(SampleTypeFlowWindowAvg)
	}

	return
}
