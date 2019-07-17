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

func (pt PanelType) String() string {
	switch pt {
	case PanelTypeInvalid:
		return "Invalid"
	case PanelTypeLindsay:
		return "Lindsay"
	case PanelTypeValleyIconSerial:
		return "Val Icon"
	case PanelTypeValleyCam:
		return "Val CAM"
	case PanelTypeReserved:
		return "Reserved"
	case PanelTypeStandardPump:
		return "Std Pump"
	case PanelTypeStandardPivot:
		return "Std Pivot"
	default:
		return "Unknown"
	}
}

// PanelDefinition is used to describe the panel.
type PanelDefinition struct {
	Voltage float64
	Type    PanelType
}
