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
	PanelTypeStandardPump
	PanelTypeStandardPivot
)

// PanelDefinition is used to describe the panel. Voltage is the uppler limit of
// the voltage range so the idea is you can loop through the definitions starting
// at the lower voltage, and simply check if V is less than the upper limit to
// identify a panel.
type PanelDefinition struct {
	Voltage     float64
	Type        PanelType
	Description string
}
