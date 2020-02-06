package isdata

import (
	"strconv"
)

var currentConfigVersion = 1

// Config represents configuration data for the Injectory
// Sentry system.
type Config struct {
	Version int

	// ID is an alphanumeric name limitted to 16 chars in length
	ID string

	// Timezone is used to store the current timezone so that the system zone can be set
	// after the timezone is edited by the user and so the timezone can be set if it is
	// erased in a system update or otherwise
	Timezone string

	// FlowRateTarget is set by pressing the arm switch
	FlowRateTarget float64

	// PressureShutdownEnabled allows users to disable the pressure shutdown functionality
	PressureShutdownEnabled bool

	// PressureShutdownLow is the lower bound set by pressing the arm switch
	PressureShutdownLow float64

	// PressureStartupLow is the minimum pressure required to arm the system
	PressureStartupLow int

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

	// LowPresPerc is the percent lower than the current pressure min necessary to shutdown the system
	LowPresPerc float64

	// HighPres is a limit that, if tripped, will immediately shutdown the irrigator in
	// monitor and shutdown mode
	HighPres int

	// This is the time in seconds until the system recognizes flow off target
	AlarmRecognizeSec float64

	// This is the time in minutes until the system activates the shutdown
	// relay after recognizing that the irrigator input is off
	// *** NOT USED ***
	IrrigatorOffMin float64

	// BatchAmount max value is 9,999
	BatchAmount         int
	WaterOn             bool
	Arm                 bool
	OperatingMode       ISOperatingMode
	CurrentFieldIndex   int // location of current (active) field
	FieldConfigs        []FieldConfig
	CurrentProductIndex int
	ProductConfigs      []ProductConfig
	DeviceName          string

	NetworkConfig   NetworkConfig
	TankCapacity    int
	TankAlertVolume int  // UNUSED
	TankAlertOn     bool // UNUSED
	FlowMeterPPG    int  // how many pulses in one US gallon
	FlowMeterMaxflo int  // Meter's maximum flow rate in GPM or LPM

	// Logging options
	LogPulseData    bool
	LogFlowData     bool
	LogPressureData bool

	// Diagnostics/Config outputs
	ManualRelayInj      RelayControlStateType
	ManualRelayAux      RelayControlStateType
	ManualRelayShutdown RelayControlStateType

	// Flow meter pulses per gallon, flow moving average time
	// windows and percent difference, pressure setting, K-factor
	// for output pulses
	PulsesPerGallon         int
	FlowAvgWindow           int
	FlowAvgWindowLong       int
	FlowAvgWindowShortUsed  bool
	FlowAvgPercDiff         int
	PressureSetting         int
	PulseOutputK            int
	PulseOutputTestFlowRate int
	PulseOutputTestOn       bool
	// Frequency with which flow rate is calculated and pulses dumped
	// -- bucket size
	SampleDuration int
	// Maximum time with no pulses before moving averages are reset and
	// flow is zeroed
	MaxNoPulseDuration int

	UserPumpMode UserPumpMode

	PanelType PanelType

	ModemEnabled bool
}

// UserPumpMode describes the state of the pump button in UI
type UserPumpMode int

// define valid user pump modes
const (
	UserPumpModeNotSet UserPumpMode = iota
	UserPumpModeOff
	UserPumpModeOn
	UserPumpModeInj
	UserPumpModeAcc1
	UserPumpModeAcc2
)

// ISOperatingMode defines the operating mode of the system
type ISOperatingMode int

// define the possible operating modes
const (
	ISOperatingModeMonitor ISOperatingMode = iota
	ISOperatingModeMonitorAndNotify
	ISOperatingModeMonitorAndShutdown
	ISOperatingModeMonitorAndBatch
)

func (om ISOperatingMode) String() string {
	switch om {
	case ISOperatingModeMonitor:
		return "monitor"
	case ISOperatingModeMonitorAndShutdown:
		return "monitor and shutdown"
	case ISOperatingModeMonitorAndBatch:
		return "monitor and batch"
	default:
		return strconv.Itoa(int(om))
	}
}

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

// RelayControlStateType is a type for relays that
// are either in auto or manual mode
type RelayControlStateType int

// define valid RelayControlStateTypes
const (
	RelayControlStateAuto RelayControlStateType = iota
	RelayControlStateOff
	RelayControlStateOn
	RelayControlStateNone
)

// BoolVal returns true for State On and default false
func (r RelayControlStateType) BoolVal() bool {
	if r == RelayControlStateOn {
		return true
	}
	return false
}

// GetMsg returns a message to pass to
// isui/menu.AddItemAutoOffOn(...)
func (r RelayControlStateType) GetMsg() int {
	switch r {
	case RelayControlStateAuto:
		return int(RelayControlStateOff)
	case RelayControlStateOff:
		return int(RelayControlStateOn)
	case RelayControlStateOn:
		return int(RelayControlStateAuto)
	}
	return int(RelayControlStateNone)
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

// CalculateFlowWindow returns high and low bounds of the flow rate window
func (c *Config) CalculateFlowWindow() (float64, float64) {

	var highBound, lowBound float64

	// check if outside lower bound
	if c.ManualLowAlarmGPH > 0 { // if the absolute GPH is set, use it
		lowBound = c.ManualLowAlarmGPH
	} else { // otherwise, compute a lowerbound in GPH from the percentage
		// target - % * target
		lowBound = c.FlowRateTarget - c.LowWindowPerc/100*c.FlowRateTarget
	}

	// check if outside upper bound
	if c.ManualHighAlarmGPH > 0 {
		highBound = c.ManualHighAlarmGPH
	} else {
		// target + % * target
		highBound = c.FlowRateTarget + c.HighWindowPerc/100*c.FlowRateTarget
	}

	return highBound, lowBound
}

// SetFlowAvgWindows forces these values to be greater than SampleDuration
// DEPRECIATED. Replaced by ApplyBounds()
func (c *Config) SetFlowAvgWindows() {
	if c.SampleDuration <= 0 {
		c.SampleDuration = 1
	}
	if c.MaxNoPulseDuration <= c.SampleDuration*2 {
		c.MaxNoPulseDuration = c.SampleDuration * 2
	}
	if c.FlowAvgWindow < c.SampleDuration*2 {
		c.FlowAvgWindow = c.SampleDuration * 2
	}
	if c.FlowAvgWindowLong < c.FlowAvgWindow*2 {
		c.FlowAvgWindowLong = c.FlowAvgWindow * 2
	}
}

// Init is used to inialize the config
func (c *Config) Init() {
	// run migrations

	if c.Version < 1 {
		c.ModemEnabled = true
	}

	c.Version = currentConfigVersion

	// always turn off logging of pulse data -- this should be
	// initiated by user each time system starts
	c.LogPulseData = false
	c.LogFlowData = false
	c.LogPressureData = false

	// Initialize pulse output test to off
	c.PulseOutputTestOn = false

	// set relays to auto mode in case
	// power lost while relays were in manual mode
	c.ManualRelayInj = RelayControlStateAuto
	c.ManualRelayAux = RelayControlStateAuto
	c.ManualRelayShutdown = RelayControlStateAuto

	if c.DeviceName == "" {
		c.DeviceName = "InjectorSentry"
	}

	if c.Timezone == "" {
		c.Timezone = "Central"
	}

	if c.PulsesPerGallon <= 0 {
		c.PulsesPerGallon = 3785
	}

	if c.FlowAvgWindow <= 0 {
		c.FlowAvgWindow = 8
	}

	if c.FlowAvgWindowLong <= 0 {
		c.FlowAvgWindowLong = 30
	}

	if c.FlowAvgPercDiff <= 0 {
		c.FlowAvgPercDiff = 10
	}

	// Check window size
	c.SetFlowAvgWindows()

	if c.PressureSetting <= 0 {
		c.PressureSetting = 300
	}

	if c.PulseOutputK <= 0 {
		c.PulseOutputK = c.PulsesPerGallon
	}

	if c.PulseOutputTestFlowRate <= 0 {
		c.PulseOutputTestFlowRate = 37
	}

	if c.HighWindowPerc <= 0 {
		c.HighWindowPerc = 15
	}

	if c.LowWindowPerc <= 0 {
		c.LowWindowPerc = 15
	}

	if c.LowPresPerc <= 0 {
		c.LowPresPerc = 50
	}

	if c.HighPres <= 0 {
		c.HighPres = 250
	}

	if c.AlarmRecognizeSec <= 0 {
		c.AlarmRecognizeSec = 30
	}

	// If the FieldConfigs array needs initialized
	if len(c.FieldConfigs) < 30 {

		// Make sure it is clean
		c.FieldConfigs = []FieldConfig{}

		// Initialize with 30 fields
		for i := 0; i < 30; i++ {
			fieldConfig := FieldConfig{
				Description: "Field " + strconv.Itoa(i+1),
			}
			c.FieldConfigs = append(c.FieldConfigs, fieldConfig)
		}
	}

	if len(c.ProductConfigs) < 5 {
		c.ProductConfigs = []ProductConfig{
			ProductConfig{"Product 1"},
			ProductConfig{"Product 2"},
			ProductConfig{"Product 3"},
			ProductConfig{"Product 4"},
			ProductConfig{"Product 5"},
		}
	}

	if c.CurrentFieldIndex > len(c.FieldConfigs)-1 {
		c.CurrentFieldIndex = len(c.FieldConfigs) - 1
	}

	if c.CurrentProductIndex > len(c.ProductConfigs)-1 {
		c.CurrentProductIndex = len(c.ProductConfigs) - 1
	}

	if c.PanelType != PanelTypeStandardPump &&
		c.PanelType != PanelTypeStandardPivot &&
		c.PanelType != PanelTypeLindsay {
		c.PanelType = PanelTypeStandardPivot
	}
}

// ApplyBounds makes sure that all the config items are within
// reasonable bounds
func (c *Config) ApplyBounds() {
	if c.PressureStartupLow < 0 {
		c.PressureStartupLow = 0
	} else if c.PressureStartupLow > 9999 {
		c.PressureStartupLow = 9999
	}

	if c.HighWindowPerc < 0 {
		c.HighWindowPerc = 0
	} else if c.HighWindowPerc > 9999 {
		c.HighWindowPerc = 9999
	}

	if c.LowWindowPerc < 0 {
		c.LowWindowPerc = 0
	} else if c.LowWindowPerc > 9999 {
		c.LowWindowPerc = 9999
	}

	if c.ManualHighAlarmGPH < 0 {
		c.ManualHighAlarmGPH = 0
	} else if c.ManualHighAlarmGPH > 9999 {
		c.ManualHighAlarmGPH = 9999
	}

	if c.ManualLowAlarmGPH < 0 {
		c.ManualLowAlarmGPH = 0
	} else if c.ManualLowAlarmGPH > 9999 {
		c.ManualLowAlarmGPH = 9999
	}

	if c.LowPresPerc < 0 {
		c.LowPresPerc = 0
	} else if c.LowPresPerc > 9999 {
		c.LowPresPerc = 9999
	}

	if c.HighPres <= 0 {
		c.HighPres = 1
	} else if c.HighPres > 9999 {
		c.HighPres = 9999
	}

	if c.AlarmRecognizeSec < 0 {
		c.AlarmRecognizeSec = 0
	} else if c.AlarmRecognizeSec > 9999 {
		c.AlarmRecognizeSec = 9999
	}

	if c.BatchAmount < 0 {
		c.BatchAmount = 0
	} else if c.BatchAmount > 9999 {
		c.BatchAmount = 9999
	}

	if c.PulsesPerGallon <= 0 { // PulsesPerGallon can't be 0
		c.PulsesPerGallon = 1
	} else if c.PulsesPerGallon > 9999 {
		c.PulsesPerGallon = 9999
	}

	if c.SampleDuration < 1 {
		c.SampleDuration = 1
	} else if c.SampleDuration > 300 {
		c.SampleDuration = 300
	}

	if c.MaxNoPulseDuration < 2*c.SampleDuration {
		c.MaxNoPulseDuration = 2 * c.SampleDuration
	} else if c.MaxNoPulseDuration > 600 {
		c.MaxNoPulseDuration = 600
	}

	if c.FlowAvgWindow < 2*c.SampleDuration {
		c.FlowAvgWindow = 2 * c.SampleDuration
	} else if c.FlowAvgWindow > 600 {
		c.FlowAvgWindow = 600
	}

	if c.FlowAvgWindowLong < 2*c.FlowAvgWindow {
		c.FlowAvgWindowLong = 2 * c.FlowAvgWindow
	} else if c.FlowAvgWindowLong > 1200 {
		c.FlowAvgWindowLong = 1200
	}

	if c.FlowAvgPercDiff < 0 {
		c.FlowAvgPercDiff = 0
	} else if c.FlowAvgPercDiff > 9999 {
		c.FlowAvgPercDiff = 9999
	}

	if c.PressureSetting < 0 {
		c.PressureSetting = 0
	} else if c.PressureSetting > 9999 {
		c.PressureSetting = 9999
	}

	if c.PulseOutputK <= 0 { // PulseOutputK can't be 0
		c.PulseOutputK = 1
	} else if c.PulseOutputK > 9999 {
		c.PulseOutputK = 9999
	}

	if c.PulseOutputTestFlowRate < 0 {
		c.PulseOutputTestFlowRate = 0
	} else if c.PulseOutputTestFlowRate > 9999 {
		c.PulseOutputTestFlowRate = 9999
	}
}
