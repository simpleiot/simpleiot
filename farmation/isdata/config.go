package isdata

import (
	"log"
	"strconv"
)

// Config represents configuration data for the Injectory
// Sentry system. Note, these type is stored directly in database,
// so you can't ever change types of fields, or remove values from
// consts. It is safest to never delete fields, but rather comment
// that they are no longer used, so they don't accidently get
// reused in the future with a different type.
type Config struct {
	Version int `json:"version"`

	// ID is an alphanumeric name limitted to 16 chars in length
	ID string `json:"id"`

	// Timezone is used to store the current timezone so that the system zone can be set
	// after the timezone is edited by the user and so the timezone can be set if it is
	// erased in a system update or otherwise
	Timezone string `json:"timezone"`

	// FlowRateTarget is set by pressing the arm switch
	FlowRateTarget float64 `json:"flowRateTarget"`

	// PressureShutdownEnabled allows users to disable the pressure shutdown functionality
	PressureShutdownEnabled bool `json:"pressureShutdownEnabled"`

	// PressureShutdownLow is the lower bound set by pressing the arm switch
	PressureShutdownLow float64 `json:"pressureShutdownLow"`

	// PressureStartupLow is the minimum pressure required to arm the system
	PressureStartupLow int `json:"pressureStartupLow"`

	// High/LowWindow will be displayed as decimal if under 10.
	// These values are % from the flow target that will trigger
	// and alarm.
	HighWindowPerc float64 `json:"highWindowPerc"`
	LowWindowPerc  float64 `json:"lowWindowPerc"`

	// ManualHigh/LowAlarm values are used to specify absolute
	// GPM values that will trigger an alarm. If these values are
	// not zero, then they will be used instead of High/LowWindowPerc
	// for triggering an alarm.
	ManualHighAlarmGPH float64 `json:"manualHighAlarmGPH"`
	ManualLowAlarmGPH  float64 `json:"manualLowAlarmGPH"`

	// LowPresPerc is the percent lower than the current pressure min necessary to shutdown the system
	LowPresPerc float64 `json:"lowPresPerc"`

	// HighPres is a limit that, if tripped, will immediately shutdown the irrigator in
	// monitor and shutdown mode
	HighPres int `json:"highPres"`

	// This is the time in seconds until the system recognizes flow off target
	AlarmRecognizeSec float64 `json:"alarmRecognizeSec"`

	// This is the time in minutes until the system activates the shutdown
	// relay after recognizing that the irrigator input is off
	// *** NOT USED ***
	IrrigatorOffMin float64 `json:"irrigatorOffMin"`

	// BatchAmount max value is 9,999
	BatchAmount         int             `json:"batchAmount"`
	WaterOn             bool            `json:"waterOn"`
	Arm                 bool            `json:"arm"`
	OperatingMode       ISOperatingMode `json:"operatingMode"`
	CurrentFieldIndex   int             `json:"currentFieldIndex"` // location of current (active) field
	FieldConfigs        []FieldConfig   `json:"fieldConfigs"`
	CurrentProductIndex int             `json:"currentProductIndex"`
	ProductConfigs      []ProductConfig `json:"productConfigs"`
	DeviceName          string          `json:"deviceName"`

	NetworkConfig   NetworkConfig `json:"networkConfig"`
	TankCapacity    int           `json:"tankCapacity"`
	TankAlertVolume int           `json:"tankAlertVolume"` // UNUSED
	TankAlertOn     bool          `json:"tankAlertOn"`     // UNUSED
	FlowMeterPPG    int           `json:"flowMeterPPG"`    // how many pulses in one US gallon
	FlowMeterMaxflo int           `json:"flowMeterMaxflo"` // Meter's maximum flow rate in GPM or LPM

	// Logging options
	LogPulseData    bool `json:"logPulseData"`
	LogFlowData     bool `json:"logFlowData"`
	LogPressureData bool `json:"logPressureData"`

	// Diagnostics/Config outputs
	ManualRelayInj      RelayControlStateType `json:"manualRelayInj"`
	ManualRelayAux      RelayControlStateType `json:"manualRelayAux"`
	ManualRelayShutdown RelayControlStateType `json:"manualRelayShutdown"`
	ManualRegValve1     RelayControlStateType `json:"manualRegValve1"`
	ManualRegValve2     RelayControlStateType `json:"manualRegValve2"`

	// Flow meter pulses per gallon, flow moving average time
	// windows and percent difference, pressure setting, K-factor
	// for output pulses
	PulsesPerGallon int `json:"pulsesPerGallon"`
	//FlowAvgWindow           int  `json:"flowAvgWindow"`
	FlowAvgWindowLong int `json:"flowAvgWindowLong"`
	//FlowAvgWindowShortUsed  bool `json:"flowAvgWindowShortUsed"`
	//FlowAvgPercDiff         int  `json:"flowAvgPercDiff"`
	PressureSetting         int  `json:"pressureSetting"`
	PulseOutputK            int  `json:"pulseOutputK"`
	PulseOutputTestFlowRate int  `json:"pulseOutputTestFlowRate"`
	PulseOutputTestOn       bool `json:"pulseOutputTestOn"`
	// Frequency with which flow rate is calculated and pulses dumped
	// -- bucket size
	SampleDuration int `json:"sampleDuration"`
	// Maximum time with no pulses before moving averages are reset and
	// flow is zeroed
	MaxNoPulseDuration int `json:"maxNoPulseDuration"`

	UserPumpMode UserPumpMode `json:"userPumpMode"`

	PanelType PanelType `json:"panelType"`

	ModemEnabled bool `json:"modemEnabled"`

	HelpScreen HelpScreen `json:"helpScreen"`
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
// Note, these constants are stored in a data base, so we can't
// ever remove or reorder them.
const (
	ISOperatingModeMonitor ISOperatingMode = iota
	ISOperatingModeMonitorAndShutdown
	ISOperatingModeMonitorAndBatch
	ISOperatingModeMonitorAndNotify
)

func (om ISOperatingMode) String() string {
	switch om {
	case ISOperatingModeMonitor:
		return "monitor"
	case ISOperatingModeMonitorAndShutdown:
		return "monitor and shutdown"
	case ISOperatingModeMonitorAndBatch:
		return "monitor and batch"
	case ISOperatingModeMonitorAndNotify:
		return "monitor and notify"
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
	Description string `json:"description"`
}

// ProductConfig describes the configuration for a product
type ProductConfig struct {
	Description string `json:"description"`
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
	/*if c.FlowAvgWindow < c.SampleDuration*2 {
		c.FlowAvgWindow = c.SampleDuration * 2
	}
	if c.FlowAvgWindowLong < c.FlowAvgWindow*2 {
		c.FlowAvgWindowLong = c.FlowAvgWindow * 2
	}
	*/
}

// HelpScreen is the complete data structure used for a help screen
type HelpScreen struct {
	Active bool
	Name   string
	Text   string
}

// NEVER, EVER, REMOVE A FUNCTION FROM THIS SLICE OR ADD ONE ANYWHERE
// BUT THE END!!!
// ONLY add migration functions to the end of the migrations slice as
// necessary when new fields are added to the config that must be
// initialized to a default value other than zero or nil.
// Let me repeat: NEVER, EVER, REMOVE A FUNCTION FROM THIS SLICE!!!
var migrations = []func(*Config){
	migration0,
	migration1,
}

// Filler migration - will never run
func migration0(c *Config) {
}

func migration1(c *Config) {

	if c.DeviceName == "" {
		c.DeviceName = "InjectorSentry"
	}

	if c.Timezone == "" {
		c.Timezone = "Central"
	}

	if c.PressureStartupLow <= 0 {
		c.PressureStartupLow = 10
	}

	if c.HighWindowPerc <= 0 {
		c.HighWindowPerc = 15
	}

	if c.LowWindowPerc <= 0 {
		c.LowWindowPerc = 15
	}

	// c.ManualHighAlarmGPH defaults to 0

	// c.ManualLowAlarmGPH defaults to 0

	if c.LowPresPerc <= 0 {
		c.LowPresPerc = 50
	}

	if c.HighPres <= 0 {
		c.HighPres = 300
	}

	if c.AlarmRecognizeSec <= 0 {
		c.AlarmRecognizeSec = 360
	}

	// c.BatchAmount defaults to 0

	if c.PulsesPerGallon <= 0 {
		c.PulsesPerGallon = 22710
	}

	if c.FlowAvgWindowLong <= 0 {
		c.FlowAvgWindowLong = 20
	}

	if c.PressureSetting <= 0 {
		c.PressureSetting = 300
	}

	if c.PulseOutputK <= 0 {
		c.PulseOutputK = 1500
	}

	if c.PulseOutputTestFlowRate <= 0 {
		c.PulseOutputTestFlowRate = 80
	}

	if c.SampleDuration <= 0 {
		c.SampleDuration = 2
	}

	if c.MaxNoPulseDuration <= 0 {
		c.MaxNoPulseDuration = 4
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
			ProductConfig{"28% UAN"},
			ProductConfig{"Product 2"},
			ProductConfig{"Product 3"},
			ProductConfig{"Product 4"},
			ProductConfig{"Product 5"},
		}
	}

	if c.PanelType != PanelTypeStandardPump &&
		c.PanelType != PanelTypeStandardPivot &&
		c.PanelType != PanelTypeLindsay {
		c.PanelType = PanelTypeStandardPivot
	}
}

// Init is used to inialize the config
func (c *Config) Init(state *State) {

	// Run migrations

	// Check that the DBVersion from the state is not
	// a bogus value
	if state.DBConfig.DBVersion > len(migrations)-1 ||
		state.DBConfig.DBVersion < 0 {
		log.Println("Error running config migrations: bogus " +
			"database version from the state.")
	} else {

		// Will ONLY run if new migration(s) have been added
		// and DBVersion is less than the length of migrations
		for v, mig := range migrations {

			if v > state.DBConfig.DBVersion {
				mig(c)
				state.DBConfig.DBVersion = v
			}
		}
	}

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

	if c.CurrentFieldIndex > len(c.FieldConfigs)-1 {
		c.CurrentFieldIndex = len(c.FieldConfigs) - 1
	}

	if c.CurrentProductIndex > len(c.ProductConfigs)-1 {
		c.CurrentProductIndex = len(c.ProductConfigs) - 1
	}

	c.HelpScreen.Active = false

	// Make sure values are in a valid range
	c.ApplyBounds()

	if c.Version < 1 {
		c.ModemEnabled = true
	}

}

// ApplyBounds makes sure that all the config items are within
// reasonable bounds
func (c *Config) ApplyBounds() {
	if c.PressureStartupLow < 0 {
		c.PressureStartupLow = 0
	} else if c.PressureStartupLow > 100 {
		c.PressureStartupLow = 100
	}

	if c.HighWindowPerc < 0 {
		c.HighWindowPerc = 0
	} else if c.HighWindowPerc > 100 {
		c.HighWindowPerc = 100
	}

	if c.LowWindowPerc < 0 {
		c.LowWindowPerc = 0
	} else if c.LowWindowPerc > 100 {
		c.LowWindowPerc = 100
	}

	if c.ManualHighAlarmGPH < 0 {
		c.ManualHighAlarmGPH = 0
	} else if c.ManualHighAlarmGPH > 1000 {
		c.ManualHighAlarmGPH = 1000
	}

	if c.ManualLowAlarmGPH < 0 {
		c.ManualLowAlarmGPH = 0
	} else if c.ManualLowAlarmGPH > 1000 {
		c.ManualLowAlarmGPH = 1000
	}

	if c.LowPresPerc < 10 {
		c.LowPresPerc = 10
	} else if c.LowPresPerc > 100 {
		c.LowPresPerc = 100
	}

	if c.HighPres <= 50 {
		c.HighPres = 51
	} else if c.HighPres > 400 {
		c.HighPres = 400
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
	} else if c.PulsesPerGallon > 25000 {
		c.PulsesPerGallon = 25000
	}

	if c.SampleDuration < 1 {
		c.SampleDuration = 1
	} else if c.SampleDuration > 25 {
		c.SampleDuration = 25
	}

	if c.MaxNoPulseDuration < 2*c.SampleDuration {
		c.MaxNoPulseDuration = 2 * c.SampleDuration
	} else if c.MaxNoPulseDuration > 50 {
		c.MaxNoPulseDuration = 50
	}

	/*if c.FlowAvgWindow < 2*c.SampleDuration {
		c.FlowAvgWindow = 2 * c.SampleDuration
	} else if c.FlowAvgWindow > 600 {
		c.FlowAvgWindow = 600
	}

	if c.FlowAvgWindowLong < 2*c.FlowAvgWindow {
		c.FlowAvgWindowLong = 2 * c.FlowAvgWindow
	} else */if c.FlowAvgWindowLong > 1200 {
		c.FlowAvgWindowLong = 1200
	}

	/*

		if c.FlowAvgPercDiff < 0 {
			c.FlowAvgPercDiff = 0
		} else if c.FlowAvgPercDiff > 500{
			c.FlowAvgPercDiff = 500
		}*/

	if c.PressureSetting < 0 {
		c.PressureSetting = 0
	} else if c.PressureSetting > 500 {
		c.PressureSetting = 500
	}

	if c.PulseOutputK <= 0 { // PulseOutputK can't be 0
		c.PulseOutputK = 1
	} else if c.PulseOutputK > 9999 {
		c.PulseOutputK = 9999
	}

	if c.PulseOutputTestFlowRate < 0 {
		c.PulseOutputTestFlowRate = 0
	} else if c.PulseOutputTestFlowRate > 1000 {
		c.PulseOutputTestFlowRate = 1000
	}
}
