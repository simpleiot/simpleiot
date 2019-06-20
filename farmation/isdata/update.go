package isdata

// UpdateFieldName is a message used to update the field name in the config
type UpdateFieldName struct {
	Index int
	Name  string
}

// UpdateProductName is a message used to update the field name in the config
type UpdateProductName struct {
	Index int
	Name  string
}

// UpdateResetTotal1 is used to reset total1
type UpdateResetTotal1 struct{}

// UpdateResetTotal2 is used to reset total2
type UpdateResetTotal2 struct{}

// UpdateResetLifetime is used to reset the lifetime total
type UpdateResetLifetime struct{}

// UpdateResetProduct1 ...
type UpdateResetProduct1 struct{}

// UpdateResetProduct2 ...
type UpdateResetProduct2 struct{}

// UpdateResetProduct3 ...
type UpdateResetProduct3 struct{}

// UpdateResetProduct4 ...
type UpdateResetProduct4 struct{}

// UpdateResetProduct5 ...
type UpdateResetProduct5 struct{}

// UpdateResetFlowPulseCount is used to reset FlowPulseCount
type UpdateResetFlowPulseCount struct{}

// UpdateLogPulseEnable is used to enable/disable logging of pulse data to USB
type UpdateLogPulseEnable bool

// UpdateLogFlowEnable is used to enable/disable loging of flow data to USB
type UpdateLogFlowEnable bool

// UpdateLogPressureEnable is used to enable/disable loging of pressure data to USB
type UpdateLogPressureEnable bool

// UpdateTankAlertEnable is used to enable/disable the tank alert on/off
type UpdateTankAlertEnable bool

// UpdateGpioDigitalInjector is used to transmit GpioDigitalInjector value to app.go
type UpdateGpioDigitalInjector bool

// UpdateGpioDigitalIrrigator is used to transmit GpioDigitalIrrigator value
type UpdateGpioDigitalIrrigator bool

// UpdateGpioDigitalWaterOn is used to transmit GpioDigitalWaterOn value
type UpdateGpioDigitalWaterOn bool

// UpdateGpioDigitalIn is used to transmit GpioDigitalIn value
type UpdateGpioDigitalIn bool

// UpdateManualRelayInj is used to toggles the injector relay
type UpdateManualRelayInj int

// UpdateManualRelayAux is used to toggle the auxilary relay
type UpdateManualRelayAux int

// UpdateManualRelayShutdown is used to toggle the shutdown relay
type UpdateManualRelayShutdown int

// UpdateManualRelayAll is used to toggle all the relays to a state
type UpdateManualRelayAll int

// UpdateGpioRelayInjector updates relay
type UpdateGpioRelayInjector bool

// UpdateGpioRelayShutdown updates relay
type UpdateGpioRelayShutdown bool

// UpdateGpioRelayAux updates relay
type UpdateGpioRelayAux bool

// UpdatePulsesPerGallon is used to send new flow meter pulses/gal. config to app.go
type UpdatePulsesPerGallon int

// UpdatePressureSetting is used to send new pressure setting (number) config to app.go
type UpdatePressureSetting int

// UpdateFlowStatus is used to send an updated flow status to app.go
type UpdateFlowStatus int

// UpdateLowWindowPerc is used to send a new flow rate window low percentage
type UpdateLowWindowPerc float64

// UpdateHighWindowPerc is used to send a new flow rate window high percentage
type UpdateHighWindowPerc float64

// UpdateManualLowAlarmGPH is used to send a new flow rate window low gallons per hour
type UpdateManualLowAlarmGPH float64

// UpdateManualHighAlarmGPH is used to send a new flow rate window high GPH
type UpdateManualHighAlarmGPH float64

// UpdateAlarmRecognizeSec is used to send a new time
type UpdateAlarmRecognizeSec float64

// UpdateDevName is used to send a new device name for the IS to app.go
type UpdateDevName string

// UpdateCurrentFieldIndex is used to select a new field
type UpdateCurrentFieldIndex int

// UpdateCurrentProductIndex is used to select a new product
type UpdateCurrentProductIndex int

// UpdateOperatingMode is used to select a new operating mode
type UpdateOperatingMode int

// UpdateUserPumpMode is used to toggle the auto/off/test pump mode from the "pump" softkey on the home/status screens
type UpdateUserPumpMode int

// UpdateLedRed is used to set the led red on/off
type UpdateLedRed bool

// UpdateLedGreen is used to set the led green on/off
type UpdateLedGreen bool

// UpdateDisarm is used to disarm from iscontrol
type UpdateDisarm bool

// UpdateIrrigationShutdown updates this state for status LED use
type UpdateIrrigationShutdown bool

// UpdateDialogMessage is used to activate and display a modal dialog
type UpdateDialogMessage string

// UpdateDialogAck is used to acknowledge a modal dialog
type UpdateDialogAck struct{}

// UpdateDialogClose is used to close a dialog after it has been acked
type UpdateDialogClose struct{}
