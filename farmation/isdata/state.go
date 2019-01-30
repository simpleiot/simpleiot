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
	switch sample.ID {
	case SampleIDFlowRate:
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
	Total float32
}

// ProductState defines the state of a product
type ProductState struct {
	Total float32
}

// GpsPos represents a GPS position
type GpsPos struct {
	Time    time.Time
	Lat     float64
	Long    float64
	Fix     int
	NumSats int
}
