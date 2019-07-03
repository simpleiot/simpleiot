package isio

import (
	"testing"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

func TestGetPanelDefinition(t *testing.T) {
	if getPanelDefintion(1.3).Type != isdata.PanelTypeValleyCam {
		t.Error("expected PanelTypeValleyCam")
	}

	if getPanelDefintion(4).Type != isdata.PanelTypeInvalid {
		t.Error("expected PanelTypeInvalid")
	}
}
