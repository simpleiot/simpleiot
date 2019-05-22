package isio

import "testing"

func TestCalcPressure(t *testing.T) {
	p := CalcPressure(5, 4.5, 250)
	if p != 250 {
		t.Error("expected 250, got: ", p)
	}

	p = CalcPressure(5, 0.5, 250)
	if p != 0 {
		t.Error("expected 0, got: ", p)
	}

	p = CalcPressure(5, 2.5, 250)
	if p != 250/2 {
		t.Error("expected 125, got: ", p)
	}
}
