package isdata

// PanelType is used to indentify type type of panel the IS is connected to
type PanelType int

// define valid panel types
const (
	PanelTypeInvalid PanelType = iota
	PanelTypeLindsay
	PanelTypeValleyIconSerial
	PanelTypeValleyCam
	PanelTypeRinkySerial
	PanelTypeReserved
	PanelTypeStandardPump  // does not have irrigator input
	PanelTypeStandardPivot // has irrigator input
)

// PanelDefinition is used to describe the panel.
type PanelDefinition struct {
	Voltage     float64
	Type        PanelType
	Description string
}
