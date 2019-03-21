package isdata

import (
	"time"

	"github.com/simpleiot/simpleiot/data"
)

// State contains the current injectory sentry state.
type State struct {
	// FlowRate defines the current flow rate of the system.
	FlowRate          float64
	BatchApplied      float64
	BatchRemaining    float64
	Total1            float64
	Total2            float64
	LifetimeTotal     float64
	CurrentTankVolume float64
	NetworkState      NetworkState

	// PanelConfig will be populated based on the panel detected
	// by the sense resistor.
	PanelConfig        ISPanelConfig
	FieldStates        []FieldState
	ProductStates      []ProductState
	GpsPos             GpsPos
	FlowStatus         FlowStatus
	IrrigationShutdown bool
	ActiveFaults       []ISEvent
	Ios                []ISIo
}

// ProcessSample populates state with sample data.
func (s *State) ProcessSample(sample data.Sample) {
	switch sample.Type {
	case SampleTypeFlowRate:
		s.FlowRate = sample.Value
	}
}

// FlowStatus describes the overall system of flow control
type FlowStatus int

// possible system status values
const (
	FlowStatusArmedOk FlowStatus = iota
	FlowStatusOffTarget
)

// FieldState describes the state of a field.
type FieldState struct {
	Total float64
}

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

// InitState initializes the field state
func InitState(s *State) {
	for len(s.FieldStates) < 4 { //not sure that 4 is the right length
		s.FieldStates = append(s.FieldStates, FieldState{0})
	}
	for len(s.ProductStates) < 5 {
		s.ProductStates = append(s.ProductStates, ProductState{0})
	}
}
