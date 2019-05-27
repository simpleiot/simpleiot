package isdata

// UpdateFieldName is a message used to update the field name in the config
type UpdateFieldName struct {
	Index int
	Name  string
}

// UpdateResetTotal1 is used to reset total1
type UpdateResetTotal1 struct{}

// UpdateResetTotal2 is used to reset total2
type UpdateResetTotal2 struct{}

// UpdateLogPulseEnable is used to enable/disable logging of pulse data to USB
type UpdateLogPulseEnable bool

// UpdateLogFlowEnable is used to enable/disable loging of flow data to USB
type UpdateLogFlowEnable bool

// UpdateLogPressureEnable is used to enable/disable loging of pressure data to USB
type UpdateLogPressureEnable bool

// UpdateTankAlertEnable is used to enable/disable the tank alert on/off
type UpdateTankAlertEnable bool

// UpdateRelayDigitalIn
