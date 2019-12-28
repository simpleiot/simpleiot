package isdata

import "time"

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

// UpdateResetTotal1 is a command to reset total1
type UpdateResetTotal1 struct{}

// UpdateResetTotal2 is a command to reset total2
type UpdateResetTotal2 struct{}

// UpdateResetLifetime is a command to reset the lifetime total
type UpdateResetLifetime struct{}

// UpdateResetCurrentProduct ...
type UpdateResetCurrentProduct struct{}

// UpdateResetFlowPulseCount is a command to reset FlowPulseCount
type UpdateResetFlowPulseCount struct{}

// UpdateDisarm is a command to disarm the system
type UpdateDisarm struct{}

// UpdateFaultActiveClearAll is a command to clear all **active** faults
type UpdateFaultActiveClearAll struct{}

// UpdateLogPulseEnable is used to enable/disable logging of pulse data to USB
type UpdateLogPulseEnable bool

// UpdateLogFlowEnable is used to enable/disable loging of flow data to USB
type UpdateLogFlowEnable bool

// UpdateLogPressureEnable is used to enable/disable loging of pressure data to USB
type UpdateLogPressureEnable bool

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

// UpdatePressureShutdownEnabled is used to enable/disable pressure-initiated shutdown
type UpdatePressureShutdownEnabled struct{}

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

// UpdateFlowAvgWindow is used to update the flow window averaging window (seconds)
type UpdateFlowAvgWindow int

// UpdateFlowAvgWindowLong is used to update the flow window long averaging window (seconds)
type UpdateFlowAvgWindowLong int

// UpdateFlowAvgPercDiff is used to update the flow window percent difference
type UpdateFlowAvgPercDiff int

// UpdatePressureSetting is used to send new pressure setting (number) config to app.go
type UpdatePressureSetting int

// UpdatePulseOutputK is used to send new setting to app.go
type UpdatePulseOutputK int

// UpdatePulseOutputTestOn is used to toggle the pulse test output
type UpdatePulseOutputTestOn bool

// UpdatePulseOutputTestFlowRate is used to update this value
type UpdatePulseOutputTestFlowRate int

// UpdateSampleDuration is used to set this value
type UpdateSampleDuration int

// UpdateMaxNoPulseDuration is used to set this value
type UpdateMaxNoPulseDuration int

// UpdateFlowStatus is used to send an updated flow status to app.go
type UpdateFlowStatus int

// UpdateLowWindowPerc is used to send a new flow rate window low percentage
type UpdateLowWindowPerc float64

// UpdateHighWindowPerc is used to send a new flow rate window high percentage
type UpdateHighWindowPerc float64

// UpdateManualLowAlarmGPH is used to send a new flow rate window low gallons per hour
type UpdateManualLowAlarmGPH float64

// UpdateLowPresPerc is used to send a new pressure low
type UpdateLowPresPerc float64

// UpdatePressureStartupLow is used to send a new minimum required startup pressure
type UpdatePressureStartupLow int

// UpdateManualHighAlarmGPH is used to send a new flow rate window high GPH
type UpdateManualHighAlarmGPH float64

// UpdateAlarmRecognizeSec is used to send a new time
type UpdateAlarmRecognizeSec float64

// UpdateTankAlertVolume ...
type UpdateTankAlertVolume int

// UpdateTankCapacity is used to set a new tank size from the tank menu
type UpdateTankCapacity int

// UpdateTankFull is used to set the current tank volume to the tank capacity
type UpdateTankFull struct{}

// UpdateCurrentTankVolume ...
type UpdateCurrentTankVolume int

// UpdateTankAlertEnable is used to enable/disable the tank alert on/off
type UpdateTankAlertEnable bool

// UpdateDevName is used to send a new device name for the IS to app.go
type UpdateDevName string

// UpdateEditedTime is used to update this value
type UpdateEditedTime time.Time

// UpdateTimezone is used to update this string value
type UpdateTimezone string

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

// UpdateDialogStateMachineMessage is used to activate and display a modal dialog
type UpdateDialogStateMachineMessage string

// UpdateDialogStateMachineAck is used to acknowledge a modal dialog
type UpdateDialogStateMachineAck struct{}

// UpdateDialogStateMachineClose is used to close a dialog after it has been acked
type UpdateDialogStateMachineClose struct{}

// UpdateDialogArmClose is used to close the arm error dialog
type UpdateDialogArmClose struct{}

// UpdateDialogArmInputsClose is used to close the arm error dialog
type UpdateDialogArmInputsClose struct{}

// UpdateDialogArmReqClose is used to close the arm requirements dialog
type UpdateDialogArmReqClose struct{}

// UpdateDialogAppClose is used to close the error dialog that originates from app.go
type UpdateDialogAppClose struct{}

// UpdateDialogUpdateClose is used to close the update dialog
type UpdateDialogUpdateClose struct{}

// UpdateDialogExportClose is used to close the export dialog
type UpdateDialogExportClose struct{}

// UpdateDialogInvalidPanelClose is used to close dialog
type UpdateDialogInvalidPanelClose struct{}

// UpdateDialogUnknownVisionStateClose is used to close the dialog that alerts users of
// an unknown state for the Vision panel
type UpdateDialogUnknownVisionStateClose struct{}
