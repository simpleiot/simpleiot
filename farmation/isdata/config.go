package isdata

// Config represents configuration data for the Injectory
// Sentry system.
type Config struct {
	// ID is an alphanumeric name limitted to 16 chars in length
	ID string

	// High/LowWindow will be displayed as decimal if under 10.
	// These values are % from the flow target that will trigger
	// and alarm.
	HighWindowPerc float64
	LowWindowPerc  float64

	// ManualHigh/LowAlarm values are used to specify absolute
	// GPM values that will trigger an alarm. If these values are
	// not zero, then they will be used instead of High/LowWindowPerc
	// for triggering an alarm.
	ManualHighAlarmGPH float64
	ManualLowAlarmGPH  float64

	// BatchAmount max value is 9,999
	BatchAmount         int
	WaterOn             bool
	Arm                 bool
	OperatingMode       ISOperatingMode
	CurrentFieldIndex   int
	FieldConfigs        []FieldConfig
	CurrentProductIndex int
	ProductConfigs      []ProductConfig
	DeviceName          string
	NetworkConfig       NetworkConfig
	TankCapacity        int
	TankAlertVolume     int
	TankAlertOn         bool
	FlowMeterPPG        int // how many pulses in one US gallon
	FlowMeterMaxflo     int // Meter's maximum flow rate in GPM or LPM

	// Logging options
	LogPulseData    bool
	LogFlowData     bool
	LogPressureData bool

	// Diagnostics/Config outputs
	ManualRelayInj      RelayControlStateType
	ManualRelayAux      RelayControlStateType
	ManualRelayShutdown RelayControlStateType

	// Flow meter pulses per gallon and pressure setting
	PulsesPerGallon int
	PressureSetting int
}

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

// FieldConfig describes the configuration for a field
type FieldConfig struct {
	Description string
}

// ProductConfig describes the configuration for a product
type ProductConfig struct {
	Description string
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
	Value       float64
}

// RelayControlStateType is a type for relays that
// are either in auto or manual mode
type RelayControlStateType int

// define valid RelayControlStateTypes
const (
	RelayControlAutoStateType RelayControlStateType = iota
	RelayControlOffStateType
	RelayControlOnStateType
	RelayControlNoneStateType
)

// BoolVal returns a boolean depending what state
// the relay is in
func (r RelayControlStateType) BoolVal() bool {
	switch r {
	case RelayControlOnStateType:
		return true
	default:
		return false
	}
}

// GetMsg returns a message to pass to
// isui/menu.AddItemAutoOffOn(...)
func (r RelayControlStateType) GetMsg() int {
	switch r {
	case RelayControlAutoStateType:
		return int(RelayControlOffStateType)
	case RelayControlOffStateType:
		return int(RelayControlOnStateType)
	case RelayControlOnStateType:
		return int(RelayControlAutoStateType)
	}
	return int(RelayControlNoneStateType)
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

// Init is used to inialize the config
func (c *Config) Init() {
	// always turn off logging of pulse data -- this should be
	// initiated by user each time system starts
	c.LogPulseData = false
	c.LogFlowData = false
	c.LogPressureData = false

	// set relays to auto mode in case
	// power lost while relays were in manual mode
	c.ManualRelayInj = RelayControlAutoStateType
	c.ManualRelayAux = RelayControlAutoStateType
	c.ManualRelayShutdown = RelayControlAutoStateType

	if len(c.DeviceName) == 0 {
		c.DeviceName = "InjectorSentry"
	}

	if c.PulsesPerGallon <= 0 {
		c.PulsesPerGallon = 3785
	}

	if c.PressureSetting <= 0 {
		c.PressureSetting = 250
	}

	if c.HighWindowPerc <= 0 {
		c.HighWindowPerc = 15
	}

	if c.LowWindowPerc <= 0 {
		c.LowWindowPerc = 15
	}

	if len(c.FieldConfigs) < 4 {
		c.FieldConfigs = []FieldConfig{
			FieldConfig{"Field One"},
			FieldConfig{"Field Two"},
			FieldConfig{"Field Three"},
			FieldConfig{"Field Four"},
		}
	}
	if len(c.ProductConfigs) < 4 {
		c.ProductConfigs = []ProductConfig{
			ProductConfig{"Product One"},
			ProductConfig{"Product Two"},
			ProductConfig{"Product Three"},
			ProductConfig{"Product Four"},
		}
	}
}
