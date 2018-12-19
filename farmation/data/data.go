package data

// ISOperatingMode defines the operating mode of the system
type ISOperatingMode int

// define the possible operating modes
const (
	ISOperatingModeMonitor = iota
	ISOperatingModeMonitorAndShutdown
	ISOperatingModeMonitorAndBatch
)

// ISConfig represents configuration data for the Injectory
// Sentry system.
type ISConfig struct {
	// ID is an alphanumeric name limitted to 16 chars in length
	ID              string
	HighWindow      float64
	LowWindow       float64
	BatchAmount     float64
	WaterOn         bool
	OperatingMode   ISOperatingMode
	ManualHighAlarm bool
	ManualLowAlarm  bool
	CurrentField    string
	Fields          []string
	NetworkConfig   NetworkConfig
}

// ISState contains the current injectory sentry state.
type ISState struct {
	// FlowRate defines the current flow rate of the system.
	FlowRate       float64
	BatchApplied   float64
	BatchRemaining float64
	Total1         float64
	Total2         float64
	NetworkState   NetworkState
}

// ISField defines a field
type ISField struct {
	Description string
	Total       float64
}

// ISProduct defines an injected product
type ISProduct struct {
	Description string
	Total       float64
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
