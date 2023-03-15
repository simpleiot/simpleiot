package client

import (
	"testing"
)

func TestShellyScanHost(t *testing.T) {
	testData := [][]string{
		{"ShellyPlugUS-C049EF8889A0.local.", "PlugUS", "C049EF8889A0"},
		{"ShellyBulbDuo-6646EB.local.", "BulbDuo", "6646EB"},
		{"shellyrgbw2-D93C00.local.", "rgbw2", "D93C00"},
		{"shelly1pm-B91754.local.", "1pm", "B91754"},
	}

	for _, e := range testData {
		typ, id := shellyScanHost(e[0])
		if typ != e[1] {
			t.Errorf("Exp: %v, got: %v", e[1], id)
		}
	}
}
