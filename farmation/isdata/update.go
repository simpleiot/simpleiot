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
