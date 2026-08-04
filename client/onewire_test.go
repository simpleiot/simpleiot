package client

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// writeW1 lays out a temporary directory like the w1 sysfs tree. A device
// appears twice: once under its bus controller, which is what detection walks,
// and once in the flat device directory, which is where the temperature is
// read from. An empty temp means the attribute file is left out, which is how
// a device that cannot be read appears.
func writeW1(t *testing.T, root string, busIndex int, deviceID, temp string) {
	t.Helper()

	busDir := filepath.Join(root, "w1_bus_master"+strconv.Itoa(busIndex), deviceID)
	if err := os.MkdirAll(busDir, 0755); err != nil {
		t.Fatal("Error creating w1 bus dir:", err)
	}

	devDir := filepath.Join(root, deviceID)
	if err := os.MkdirAll(devDir, 0755); err != nil {
		t.Fatal("Error creating w1 device dir:", err)
	}

	if temp == "" {
		return
	}

	err := os.WriteFile(filepath.Join(devDir, "temperature"), []byte(temp+"\n"), 0644)
	if err != nil {
		t.Fatal("Error writing w1 temperature:", err)
	}
}

func TestOneWireDetect(t *testing.T) {
	root := t.TempDir()

	writeW1(t, root, 0, "28-000005e2fdc3", "23456")
	writeW1(t, root, 0, "28-000005e2fdc4", "23456")
	// a sensor on a second controller, which bus 0 must not claim
	writeW1(t, root, 1, "28-000005e2fdc5", "23456")

	// a non temperature device on bus 0, which is not a DS18B20
	err := os.MkdirAll(filepath.Join(root, "w1_bus_master0", "10-000802be7e9d"), 0755)
	if err != nil {
		t.Fatal("Error creating w1 device dir:", err)
	}

	tests := []struct {
		index int
		ids   []string
	}{
		{0, []string{"28-000005e2fdc3", "28-000005e2fdc4"}},
		{1, []string{"28-000005e2fdc5"}},
		// a bus controller that is not present
		{2, nil},
	}

	for _, tc := range tests {
		ids, err := oneWireDetect(root, tc.index)
		if err != nil {
			t.Fatalf("bus %v: detect error: %v", tc.index, err)
		}

		if len(ids) != len(tc.ids) {
			t.Fatalf("bus %v: expected %v devices, got %v", tc.index, tc.ids, ids)
		}

		for i, id := range tc.ids {
			if ids[i] != id {
				t.Errorf("bus %v: expected device %v, got %v", tc.index, id, ids[i])
			}
		}
	}
}

func TestOneWireRead(t *testing.T) {
	root := t.TempDir()

	writeW1(t, root, 0, "28-celsius", "23456")
	writeW1(t, root, 0, "28-negative", "-1500")
	writeW1(t, root, 0, "28-empty", " ")
	writeW1(t, root, 0, "28-garbage", "unavailable")
	// a device with no temperature attribute at all
	writeW1(t, root, 0, "28-missing", "")

	tests := []struct {
		desc     string
		deviceID string
		units    string
		value    float64
		wantErr  bool
	}{
		{"celsius", "28-celsius", "", 23.456, false},
		{"celsius explicit", "28-celsius", "C", 23.456, false},
		{"fahrenheit", "28-celsius", "F", 23.456*1.8 + 32, false},
		{"below freezing", "28-negative", "", -1.5, false},
		{"below freezing in F", "28-negative", "F", 29.3, false},
		{"empty file", "28-empty", "", 0, true},
		{"unparseable", "28-garbage", "", 0, true},
		{"missing file", "28-missing", "", 0, true},
		{"unknown device", "28-nothere", "", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			v, err := oneWireRead(root, tc.deviceID, tc.units)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got value %v", v)
				}
				return
			}

			if err != nil {
				t.Fatal("read error: ", err)
			}

			if math.Abs(v-tc.value) > 1e-9 {
				t.Fatalf("expected %v, got %v", tc.value, v)
			}
		})
	}
}

// TestOneWirePollPeriod checks the default that applies when a bus has no poll
// period set.
func TestOneWirePollPeriod(t *testing.T) {
	c := &OneWireClient{}
	if got := c.pollPeriod(); got != oneWireDefaultPollPeriod {
		t.Errorf("expected the default poll period %v, got %v",
			oneWireDefaultPollPeriod, got)
	}

	c.config.PollPeriod = 500
	if got := c.pollPeriod().Milliseconds(); got != 500 {
		t.Errorf("expected a 500ms poll period, got %v", got)
	}
}
