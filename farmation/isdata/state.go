package isdata

import "time"

// ISState contains the current injectory sentry state.
type ISState struct {
	// FlowRate defines the current flow rate of the system.
	FlowRate          float32
	BatchApplied      float32
	BatchRemaining    float32
	Total1            float32
	Total2            float32
	LifetimeTotal     float32
	CurrentTankVolume float32
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
