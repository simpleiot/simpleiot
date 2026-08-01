package client

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/simpleiot/simpleiot/data"
)

// writeZone creates a thermal zone directory in the fixture. An empty typ or
// temp means that file is left out, which is how an unreadable zone appears.
func writeZone(t *testing.T, dir, zone, typ, temp string) {
	t.Helper()

	z := filepath.Join(dir, zone)
	if err := os.MkdirAll(z, 0755); err != nil {
		t.Fatal("Error creating zone dir:", err)
	}

	if typ != "" {
		if err := os.WriteFile(filepath.Join(z, "type"), []byte(typ+"\n"), 0644); err != nil {
			t.Fatal("Error writing zone type:", err)
		}
	}

	if temp != "" {
		if err := os.WriteFile(filepath.Join(z, "temp"), []byte(temp+"\n"), 0644); err != nil {
			t.Fatal("Error writing zone temp:", err)
		}
	}
}

func TestThermalZones(t *testing.T) {
	dir := t.TempDir()

	writeZone(t, dir, "thermal_zone0", "cpu-thermal", "37718")
	// a powered down rail returns an error when its temp is read
	writeZone(t, dir, "thermal_zone1", "gpu-thermal", "")
	// a zone that reports something we cannot parse
	writeZone(t, dir, "thermal_zone2", "cv0-thermal", "unavailable")
	writeZone(t, dir, "thermal_zone3", "tj-thermal", "37781")
	// unrelated entries in the same directory are left alone
	writeZone(t, dir, "cooling_device0", "fan", "1")

	temps := thermalZones(dir)

	exp := []temperature{
		{key: "cpu-thermal", temp: 37.718},
		{key: "tj-thermal", temp: 37.781},
	}

	if len(temps) != len(exp) {
		t.Fatalf("Expected %v readings, got %v: %+v", len(exp), len(temps), temps)
	}

	for i, e := range exp {
		if temps[i] != e {
			t.Errorf("Expected reading %v to be %+v, got %+v", i, e, temps[i])
		}
	}
}

func TestThermalZonesNoInterface(t *testing.T) {
	if temps := thermalZones(filepath.Join(t.TempDir(), "does-not-exist")); len(temps) != 0 {
		t.Error("Expected no readings without a thermal interface, got:", temps)
	}
}

func TestSysTemperaturesUniqueKeys(t *testing.T) {
	dir := t.TempDir()

	// sensor names are not unique, and duplicate keys would overwrite each
	// other in the point store
	writeZone(t, dir, "thermal_zone0", "cpu-thermal", "37718")
	writeZone(t, dir, "thermal_zone1", "cpu-thermal", "38000")

	old := thermalZonePath
	thermalZonePath = dir
	defer func() { thermalZonePath = old }()

	exp := map[string]float64{
		"cpu-thermal":   37.718,
		"cpu-thermal_2": 38,
	}

	keys := make(map[string]bool)

	for _, p := range sysTemperatures() {
		if p.Type != data.PointTypeTemperature {
			t.Error("Expected temperature points, got:", p.Type)
		}

		if keys[p.Key] {
			t.Error("Duplicate temperature key:", p.Key)
		}
		keys[p.Key] = true

		if e, ok := exp[p.Key]; ok {
			if p.Val() != e {
				t.Errorf("Expected %v to be %v, got %v", p.Key, e, p.Val())
			}
			delete(exp, p.Key)
		}
	}

	for k := range exp {
		t.Error("Missing temperature reading for zone:", k)
	}
}
