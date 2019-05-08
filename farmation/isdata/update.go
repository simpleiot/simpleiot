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

// UpdateToggleLogPulse is used to toggle logging of pulse data to USB
type UpdateToggleLogPulse struct{}

// UpdateToggleLogFlow is used to toggle loging of flow data to USB
type UpdateToggleLogFlow struct{}

// UpdateToggleTankAlert is used to toggle the tank alert on/off
type UpdateToggleTankAlert struct{}
