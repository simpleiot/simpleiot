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

// UpdateGpioDigitalInjector is used to transmit GpioDigitalInjector value
type UpdateGpioDigitalInjector bool

// UpdateGpioDigitalIrrigator is used to transmit GpioDigitalIrrigator value
type UpdateGpioDigitalIrrigator bool

// UpdateGpioDigitalWaterOn is used to transmit GpioDigitalWaterOn value
type UpdateGpioDigitalWaterOn bool

// UpdateGpioDigitalIn is used to transmit GpioDigitalIn value
type UpdateGpioDigitalIn bool

// UpdateManualRelayInj is used to toggles the injector relay
type UpdateManualRelayInj bool

// UpdateManualRelayAux is used to toggle the auxilary relay
type UpdateManualRelayAux bool

// UpdateManualRelayShutdown is used to toggle the shutdown relay
type UpdateManualRelayShutdown bool
