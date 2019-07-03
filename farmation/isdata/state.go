package isdata

import (
	"runtime"
	"time"
)

// State contains the current injectory sentry state.
type State struct {
	SystemType SystemType `json:"systemType"`

	// FlowRate defines the current flow rate of the system in GPH
	FlowRate          float64      `json:"flowRate"`
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

	// PanelDefinition will be populated based on the panel detected
	// by the sense resistor.
	PanelDefinition PanelDefinition   `json:"panelConfig"`
	FieldStates     [][5]ProductState `json:"fieldStates"`
	GpsPos          GpsPos            `json:"gpsPos"`
	FlowStatus      FlowStatus        `json:"flowStatus"`
	ActiveFaults    []ISEvent         `json:"activeFaults"`
	Ios             []ISIo            `json:"ios"`
	PressureMin     float64           `json:"pressureMin"`
	PressureMax     float64           `json:"pressureMax"`
	PressureAvg     float64           `json:"pressureAvg"`
	PressureVRef    float64           `json:"pressureVRef"`
	PressureVSense  float64           `json:"pressureVSense"`

	// Gpio's
	GpioDigitalInjector  bool `json:"gpioDigitalInjector"`
	GpioDigitalIrrigator bool `json:"gpioDigitalIrrigator"`
	GpioDigitalWaterOn   bool `json:"gpioDigitalWaterOn"`
	GpioDigitalIn        bool `json:"gpioDigitalIn"`

	GpioRelayInjectorEn bool `json:"gpioRelayInjectorEn"`
	GpioRelayShutdownEn bool `json:"gpioRelayShutdownEn"`
	GpioRelayAuxEn      bool `json:"gpioRelayAuxEn"`

	GpioStatusLedRed   bool `json:"gpioStatusLedRed"`
	GpioStatusLedGreen bool `json:"gpioStatusLedGreen"`

	// Data from Lindsay panel
	LindsayRegs       LindsayStatusRegs `json:"lindsayRegs"`
	LindsayLastUpdate time.Time         `json:"lindsayLastUpdate"`

	// Faults
	FaultsActive Faults

	// Modal dialog describes a modal dialog message
	// only for messages from state machine. Create new dialog
	// structs for other parts of the app.
	DialogStateMachine Dialog

	DialogArm Dialog
}

// InputState is a type that describes if the input is avaliable, and what its state is
type InputState int

// define value input states
const (
	InputStateNA InputState = iota
	InputStateOff
	InputStateOn
)

// BoolToInputState converts a bool to input state, assuming
// it is not InputStateNA
func BoolToInputState(v bool) InputState {
	if v {
		return InputStateOn
	}

	return InputStateOff
}

// WaterOn returns water on status based on panel type
func (s *State) WaterOn() InputState {
	switch s.PanelDefinition.Type {
	case PanelTypeStandardPump, PanelTypeStandardPivot:
		return BoolToInputState(s.GpioDigitalWaterOn)
	case PanelTypeLindsay:
		return BoolToInputState(s.LindsayRegs.WaterOn())
	default:
		return InputStateNA
	}
}

// IrrigatorRunning returns if the irrigator is running based on panel type
func (s *State) IrrigatorRunning() InputState {
	switch s.PanelDefinition.Type {
	case PanelTypeStandardPump:
		return BoolToInputState(s.GpioDigitalIrrigator)
	case PanelTypeLindsay:
		return BoolToInputState(s.LindsayRegs.IrrigatorRunning())
	default:
		return InputStateNA
	}
}

// InjectorOn returns if the injector is on for various panel types
func (s *State) InjectorOn() InputState {
	switch s.PanelDefinition.Type {
	case PanelTypeStandardPump, PanelTypeStandardPivot:
		return BoolToInputState(s.GpioDigitalInjector)
	case PanelTypeLindsay:
		return BoolToInputState(s.LindsayRegs.Accessory1On())
	default:
		return InputStateNA
	}
}

// Faults defines a struct for FaultsActive
type Faults struct {
	Irrigator bool
}

// Dialog defines a modal dialog that must be acknowledged
type Dialog struct {
	Message      string
	Active       bool
	Acknowledged bool
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
	for len(s.FieldStates) < 4 { //not sure that 4 is the right length
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
	s.GpioStatusLedRed = false
	s.GpioStatusLedGreen = false

	s.DialogArm.Active = false
	s.DialogArm.Acknowledged = false

	s.PanelDefinition = PanelDefinition{Description: "Invalid"}

	// add an active fault to test
	/*if len(s.ActiveFaults) <= 0 {
		s.ActiveFaults = append(s.ActiveFaults, ISEvent{})
	}*/

	// empty active faults
	// s.ActiveFaults = nil

	return
}
