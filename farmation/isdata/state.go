package isdata

import (
	"time"
)

// State contains the current injectory sentry state.
type State struct {
	// FlowRate defines the current flow rate of the system in GPH
	FlowRate          float64
	BatchApplied      float64
	BatchRemaining    float64
	Total1            float64
	Total2            float64
	LifetimeTotal     float64
	FlowPulseCount    int
	CurrentTankVolume float64
	NetworkState      NetworkState

	// PanelConfig will be populated based on the panel detected
	// by the sense resistor.
	PanelConfig        ISPanelConfig
	FieldStates        [][5]ProductState
	GpsPos             GpsPos
	FlowStatus         FlowStatus
	IrrigationShutdown bool
	ActiveFaults       []ISEvent
	Ios                []ISIo
	PressureMin        float64
	PressureMax        float64
	PressureAvg        float64
	PressureVRef       float64
	PressureVSense     float64

	// Gpio's
	GpioDigitalInjector  bool
	GpioDigitalIrrigator bool
	GpioDigitalWaterOn   bool
	GpioDigitalIn        bool
}

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

	s.PressureMin = 0
	s.PressureAvg = 0
	s.PressureMax = 0
	s.PressureVRef = 0
	s.PressureVSense = 0

	// add an active fault to test
	/*if len(s.ActiveFaults) <= 0 {
		s.ActiveFaults = append(s.ActiveFaults, ISEvent{})
	}*/

	// empty active faults
	// s.ActiveFaults = nil

	return
}
