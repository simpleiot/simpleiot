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

	// PanelConfig will be populated based on the panel detected
	// by the sense resistor.
	PanelConfig        ISPanelConfig     `json:"panelConfig"`
	FieldStates        [][5]ProductState `json:"fieldStates"`
	GpsPos             GpsPos            `json:"gpsPos"`
	FlowStatus         FlowStatus        `json:"flowStatus"`
	IrrigationShutdown bool              `json:"irrigationShutdown"`
	ActiveFaults       []ISEvent         `json:"activeFaults"`
	Ios                []ISIo            `json:"ios"`
	PressureMin        float64           `json:"pressureMin"`
	PressureMax        float64           `json:"pressureMax"`
	PressureAvg        float64           `json:"pressureAvg"`
	PressureVRef       float64           `json:"pressureVRef"`
	PressureVSense     float64           `json:"pressureVSense"`

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

	s.GpioRelayInjectorEn = true
	s.GpioRelayShutdownEn = false
	s.GpioRelayAuxEn = false
	s.GpioStatusLedRed = false
	s.GpioStatusLedGreen = false

	// add an active fault to test
	/*if len(s.ActiveFaults) <= 0 {
		s.ActiveFaults = append(s.ActiveFaults, ISEvent{})
	}*/

	// empty active faults
	// s.ActiveFaults = nil

	return
}
