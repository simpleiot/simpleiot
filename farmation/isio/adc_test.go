package isio

import "testing"

func TestGetPanelDefinition(t *testing.T) {
	if getPanelDefintion(1.3).Type != PanelTypeValleyCam {
		t.Error("expected PanelTypeValleyCam")
	}

	if getPanelDefintion(4).Type != PanelTypeInvalid {
		t.Error("expected PanelTypeInvalid")
	}
}
