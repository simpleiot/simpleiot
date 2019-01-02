package isdata

import "time"

// ISOperatingMode defines the operating mode of the system
type ISOperatingMode int

// define the possible operating modes
const (
	ISOperatingModeMonitor ISOperatingMode = iota
	ISOperatingModeMonitorAndShutdown
	ISOperatingModeMonitorAndBatch
)

// InjectorPumpMode describes the current state of the
// injector pump control (selected by the pump key)
type InjectorPumpMode int

// define possible injector pump modes
const (
	// In DigIn mode, the pump is controlled by a digital input.
	InjectorPumpModeDigIn InjectorPumpMode = iota

	// In manual mode, any time the water input is on, the
	// pump will be on.
	InjectorPumpModeManual

	// In aux modes, the aux1 or aux2 serial inputs are used to
	// control the pump.
	InjectorPumpModeAux1
	InjectorPumpModeAux2

	// In off mode, the pump is always off.
	InjectorPumpModeOff

	// Pump is running in test mode for 60s, and will return to
	// Off state when finished, or if user presses any key.
	InjectorPumpModeTestRun
)

// ISConfig represents configuration data for the Injectory
// Sentry system.
type ISConfig struct {
	// ID is an alphanumeric name limitted to 16 chars in length
	ID string

	// High/LowWindow will be displayed as decimal if under 10.
	HighWindow float32
	LowWindow  float32

	// BatchAmount max value is 9,999
	BatchAmount     int
	WaterOn         bool
	OperatingMode   ISOperatingMode
	ManualHighAlarm bool
	ManualLowAlarm  bool
	CurrentField    string
	FieldConfigs    []FieldConfig
	ProductConfigs  []ProductConfig
	NetworkConfig   NetworkConfig
	MaxTankVolume   int
	TankAlertVolume int
	FlowMeterPPG    int // how many pulses in one US gallon
	FlowMeterMaxflo int // Meter's maximum flow rate in GPM or LPM
}

// ISConfigDefault contains defaults for initializing a new system
var ISConfigDefault = ISConfig{
	ID:              "",
	HighWindow:      0,
	LowWindow:       0,
	BatchAmount:     0,
	WaterOn:         false,
	ManualHighAlarm: false,
	MaxTankVolume:   0,
	TankAlertVolume: 0,
}

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

// FieldConfig describes the configuration for a field
type FieldConfig struct {
	Description string
}

// FieldState describes the state of a field.
type FieldState struct {
	Total float32
}

// ProductConfig describes the configuration for a product
type ProductConfig struct {
	Description string
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

// SerialType defines the type of serial communication
type SerialType int

// Define valid serial types
const (
	SerialTypeNone = iota
	SerialTypeRS485
	SerialTypeRS232
	SerialTypeCAN
)

// IOType defines various IO types for a function
type IOType int

// define valid IOTypes
const (
	IOTypeNA IOType = iota
	IOTypeSerial
	IOTypeDigIn
)

// ISPanelConfig defines a panel the IS is connected to
type ISPanelConfig struct {
	ADCValue         float64
	Description      string
	SerialType       SerialType
	IrrigatorRunning IOType
	WaterOn          IOType
	InjectorOn       IOType
	Position         IOType
	Direction        IOType
	PowerCoControl   IOType
	Aux1             IOType
	Aux2             IOType
}

// PanelConfigs describes all of the currently supported
// panels. Based on the ADCValue, the appropriate config will
// be selected and populated in the ISState
var PanelConfigs = []ISPanelConfig{
	ISPanelConfig{
		ADCValue:         117,
		Description:      "Lindsay Vision and Boss",
		SerialType:       SerialTypeRS485,
		IrrigatorRunning: IOTypeSerial,
		WaterOn:          IOTypeSerial,
		InjectorOn:       IOTypeSerial,
		Position:         IOTypeSerial,
		Direction:        IOTypeSerial,
		PowerCoControl:   IOTypeSerial,
		Aux1:             IOTypeSerial,
		Aux2:             IOTypeSerial,
	},
	ISPanelConfig{
		ADCValue:         224,
		Description:      "Valley Icon serial",
		SerialType:       SerialTypeRS232,
		IrrigatorRunning: IOTypeSerial,
		WaterOn:          IOTypeSerial,
		InjectorOn:       IOTypeSerial,
		Position:         IOTypeSerial,
		Direction:        IOTypeSerial,
		PowerCoControl:   IOTypeNA,
		Aux1:             IOTypeSerial,
		Aux2:             IOTypeSerial,
	},
	ISPanelConfig{
		ADCValue:         340,
		Description:      "Valley CAM Panel",
		SerialType:       SerialTypeRS232,
		IrrigatorRunning: IOTypeSerial,
		WaterOn:          IOTypeSerial,
		InjectorOn:       IOTypeSerial,
		Position:         IOTypeSerial,
		Direction:        IOTypeSerial,
		PowerCoControl:   IOTypeNA,
		Aux1:             IOTypeNA,
		Aux2:             IOTypeNA,
	},
	ISPanelConfig{
		ADCValue:         799,
		Description:      "Standard Pump Panel",
		SerialType:       SerialTypeNone,
		IrrigatorRunning: IOTypeNA,
		WaterOn:          IOTypeDigIn,
		InjectorOn:       IOTypeDigIn,
		Position:         IOTypeNA,
		Direction:        IOTypeNA,
		PowerCoControl:   IOTypeNA,
		Aux1:             IOTypeNA,
		Aux2:             IOTypeNA,
	},
	ISPanelConfig{
		ADCValue:         913,
		Description:      "Standard Pivot Panel",
		SerialType:       SerialTypeNone,
		IrrigatorRunning: IOTypeDigIn,
		WaterOn:          IOTypeDigIn,
		InjectorOn:       IOTypeDigIn,
		Position:         IOTypeNA,
		Direction:        IOTypeNA,
		PowerCoControl:   IOTypeNA,
		Aux1:             IOTypeNA,
		Aux2:             IOTypeNA,
	},
}

// IOs

// ISIoType defines various IO types
type ISIoType int

// define valid ISIoTypes
const (
	ISIoType4to20In ISIoType = iota
	ISIoTypeAnalogIn
	ISIoTypeDigIn
	ISIoTypePwmOut
	ISIoTypePulseIn
	ISIotype4to20Out
)

// ISIo holds IO attributes
type ISIo struct {
	Type        ISIoType
	Description string
	Fault       bool
	Value       float32
}

// RelayID identifies a relay in the system
type RelayID int

// define possible relay IDs
const (
	// Shutdown relay will be used to shut down the
	// Irrigation system.
	RelayIDShutdown RelayID = iota

	// Injector relay is used to enable/disable the
	// injector pump.
	RelayIDInjector

	// Aux relay is for future use.
	RelayIDAux
)

// ISRelay describes the current relay state
type ISRelay struct {
	ID    RelayID
	On    bool
	Fault bool
}

// Key defines keypad inputs
type Key int

// define valid keys
const (
	KeyUp Key = iota
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeySK1
	KeySK2
	KeySK3
	KeySK4
)

func (k Key) String() string {
	switch k {
	case KeyUp:
		return "KeyUp"
	case KeyDown:
		return "KeyDown"
	case KeyLeft:
		return "KeyLeft"
	default:
		return "unkown key"

	}
}

// Screen defines various screens that can be displayed
type Screen struct {
}

// NetworkInterface defines the current network interface
type NetworkInterface int

// define valid network interfaces
const (
	NetworkInterfaceEthernet NetworkInterface = iota
	NetworkInterfaceCellular
)

// NetworkConfig defines the current network configuration
type NetworkConfig struct {
	Interface  NetworkInterface
	DHCP       bool
	StaticIP   string
	SubnetMask string
	DefaultGw  string
}

// NetworkState defines the current network state
type NetworkState struct {
	EthernetDetected bool
	ModemDetected    bool
	Connected        bool
	ModemRSRQ        float64
}

// ISEvent defines various IS Events that can occur
type ISEvent struct {
}
