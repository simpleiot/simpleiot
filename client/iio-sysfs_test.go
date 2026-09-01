package client

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeIIOFixture lays out one device directory under root from a map of
// attribute names to contents. An empty root device directory is enough for a
// device with no channels.
func writeIIOFixture(t *testing.T, root, dir string, attrs map[string]string) string {
	t.Helper()

	path := filepath.Join(root, dir)
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal("Error creating IIO device dir:", err)
	}

	for name, content := range attrs {
		err := os.WriteFile(filepath.Join(path, name), []byte(content+"\n"), 0644)
		if err != nil {
			t.Fatalf("Error writing IIO attribute %v: %v", name, err)
		}
	}

	return path
}

func TestIIOParseChannel(t *testing.T) {
	tests := []struct {
		name   string
		typ    string
		output bool
		ok     bool
	}{
		{"in_voltage0", "voltage", false, true},
		{"in_voltage0-voltage1", "voltage", false, true},
		{"in_accel_x", "accel", false, true},
		{"in_temp", "temp", false, true},
		{"in_humidityrelative", "humidityrelative", false, true},
		{"out_voltage0", "voltage", true, true},
		{"out_altvoltage0", "altvoltage", true, true},
		// not a channel prefix at all
		{"name", "", false, false},
		{"sampling_frequency", "", false, false},
		{"in_0", "", false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			typ, output, ok := iioParseChannel(tc.name)

			if ok != tc.ok {
				t.Fatalf("expected ok %v, got %v", tc.ok, ok)
			}

			if !ok {
				return
			}

			if typ != tc.typ {
				t.Errorf("expected type %v, got %v", tc.typ, typ)
			}

			if output != tc.output {
				t.Errorf("expected output %v, got %v", tc.output, output)
			}
		})
	}
}

func TestIIOFind(t *testing.T) {
	root := t.TempDir()

	writeIIOFixture(t, root, "iio:device0", map[string]string{"name": "ads1015"})
	path1 := writeIIOFixture(t, root, "iio:device1", map[string]string{"name": "lsm6dsl"})
	// a device whose driver publishes no name attribute
	writeIIOFixture(t, root, "iio:device2", nil)

	tests := []struct {
		desc    string
		device  string
		path    string
		name    string
		wantErr bool
	}{
		{"by name", "lsm6dsl", path1, "lsm6dsl", false},
		{"by directory", "iio:device0", filepath.Join(root, "iio:device0"), "ads1015", false},
		{"by path", path1, path1, "lsm6dsl", false},
		{"no name attribute", "iio:device2", filepath.Join(root, "iio:device2"), "", false},
		{"not present", "ads1115", "", "", true},
		{"nothing set", "", "", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			dev, err := iioFind(root, tc.device)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %+v", dev)
				}
				return
			}

			if err != nil {
				t.Fatal("find error:", err)
			}

			if dev.Path != tc.path {
				t.Errorf("expected path %v, got %v", tc.path, dev.Path)
			}

			if dev.Name != tc.name {
				t.Errorf("expected name %v, got %v", tc.name, dev.Name)
			}
		})
	}
}

func TestIIOChannels(t *testing.T) {
	root := t.TempDir()

	path := writeIIOFixture(t, root, "iio:device0", map[string]string{
		"name":               "ads1015",
		"sampling_frequency": "1600",
		"in_voltage0_raw":    "1583",
		"in_voltage0_scale":  "2.0",
		"in_voltage1_raw":    "1584",
		"in_temp_raw":        "2634",
		// a channel published only as a converted value
		"in_pressure_input": "101.325",
		// the same channel published both ways is listed once
		"in_humidityrelative_raw":   "512",
		"in_humidityrelative_input": "48.5",
		"out_voltage0_raw":          "2048",
	})

	chans, err := iioChannels(iioDevice{Path: path})
	if err != nil {
		t.Fatal("channels error:", err)
	}

	want := []iioChannelInfo{
		{"in_humidityrelative", "humidityrelative", false},
		{"in_pressure", "pressure", false},
		{"in_temp", "temp", false},
		{"in_voltage0", "voltage", false},
		{"in_voltage1", "voltage", false},
		{"out_voltage0", "voltage", true},
	}

	if len(chans) != len(want) {
		t.Fatalf("expected %v channels, got %+v", len(want), chans)
	}

	for i, w := range want {
		if chans[i] != w {
			t.Errorf("channel %v: expected %+v, got %+v", i, w, chans[i])
		}
	}
}

func TestIIOReadSysfs(t *testing.T) {
	root := t.TempDir()

	path := writeIIOFixture(t, root, "iio:device0", map[string]string{
		// a raw count with a per channel scale and no offset
		"in_voltage0_raw":   "1583",
		"in_voltage0_scale": "2.0",
		// a channel with no scale of its own, which falls back to the per
		// type attribute
		"in_voltage1_raw":  "1000",
		"in_voltage_scale": "3.0",
		// a raw count with no scale and no offset at all, on a type that has
		// no per type scale either
		"in_proximity_raw": "1250",
		// the offset is applied before the scale
		"in_temp_raw":    "2634",
		"in_temp_scale":  "0.062500000",
		"in_temp_offset": "-1092",
		// a converted value the client prefers over the raw count
		"in_pressure_raw":   "1",
		"in_pressure_scale": "1000",
		"in_pressure_input": "101.325",
		// a type the ABI publishes in a base unit already
		"in_accel_x_raw":   "16384",
		"in_accel_x_scale": "0.000598",
		// a value that cannot be parsed
		"in_illuminance_raw": "unavailable",
	})

	dev := iioDevice{Path: path}

	tests := []struct {
		desc    string
		channel string
		value   float64
		wantErr bool
	}{
		{"millivolts to volts", "in_voltage0", 3.166, false},
		{"per type scale", "in_voltage1", 3.0, false},
		{"no scale or offset", "in_proximity", 1250, false},
		{"offset before scale", "in_temp", (2634 - 1092) * 0.0625 / 1000, false},
		{"input preferred over raw", "in_pressure", 101.325, false},
		{"already a base unit", "in_accel_x", 16384 * 0.000598, false},
		{"unparseable", "in_illuminance", 0, true},
		{"missing channel", "in_voltage9", 0, true},
		{"not a channel name", "sampling_frequency", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			v, err := iioRead(dev, tc.channel)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got value %v", v)
				}
				return
			}

			if err != nil {
				t.Fatal("read error:", err)
			}

			if math.Abs(v-tc.value) > 1e-9 {
				t.Fatalf("expected %v, got %v", tc.value, v)
			}
		})
	}
}

func TestIIOWriteSysfs(t *testing.T) {
	root := t.TempDir()

	path := writeIIOFixture(t, root, "iio:device0", map[string]string{
		"out_voltage0_raw":    "0",
		"out_voltage0_scale":  "2.0",
		"out_voltage1_raw":    "0",
		"out_voltage1_scale":  "2.0",
		"out_voltage1_offset": "-100",
		"out_voltage2_raw":    "0",
		"out_voltage2_scale":  "0",
	})

	dev := iioDevice{Path: path}

	tests := []struct {
		desc    string
		channel string
		value   float64
		raw     string
		wantErr bool
	}{
		// 1.5 V is 1500 mV, which at a scale of 2 is a count of 750
		{"volts to a raw count", "out_voltage0", 1.5, "750", false},
		// the offset the kernel adds on the way in is removed on the way out
		{"offset removed", "out_voltage1", 1.5, "850", false},
		{"zero scale", "out_voltage2", 1.5, "", true},
		{"not a channel name", "sampling_frequency", 1, "", true},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := iioWrite(dev, tc.channel, tc.value)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}

			if err != nil {
				t.Fatal("write error:", err)
			}

			got, err := iioReadAttr(path, tc.channel+"_raw")
			if err != nil {
				t.Fatal("error reading back:", err)
			}

			if got != tc.raw {
				t.Fatalf("expected raw %v, got %v", tc.raw, got)
			}
		})
	}
}

func TestIIOWriteAttr(t *testing.T) {
	root := t.TempDir()

	path := writeIIOFixture(t, root, "iio:device0", map[string]string{
		"sampling_frequency": "1600",
	})

	dev := iioDevice{Path: path}

	if err := iioWriteAttr(dev, "sampling_frequency", "128"); err != nil {
		t.Fatal("error writing sampling_frequency:", err)
	}

	got, err := iioReadAttr(path, "sampling_frequency")
	if err != nil {
		t.Fatal("error reading back sampling_frequency:", err)
	}

	if got != "128" {
		t.Errorf("expected sampling_frequency 128, got %v", got)
	}

	// a setting the driver does not publish is reported distinctly, so that
	// an unsupported device setting is not counted as a failure
	err = iioWriteAttr(dev, "oversampling_ratio", "4")
	if !errors.Is(err, errIIOAttrMissing) {
		t.Errorf("expected a missing attribute error, got %v", err)
	}
}

// TestIIOPollPeriod checks the default that applies when a device has no poll
// period set.
func TestIIOPollPeriod(t *testing.T) {
	c := &IIOClient{}
	if got := c.pollPeriod(); got != iioDefaultPollPeriod {
		t.Errorf("expected the default poll period %v, got %v",
			iioDefaultPollPeriod, got)
	}

	c.config.PollPeriod = 500
	if got := c.pollPeriod().Milliseconds(); got != 500 {
		t.Errorf("expected a 500ms poll period, got %v", got)
	}
}

// TestIIOShouldPublish covers the decision that keeps ADC noise out of the
// store: a value must move by at least minChange, with the refresh underneath
// it so an unchanged reading is still republished.
func TestIIOShouldPublish(t *testing.T) {
	c := &IIOClient{lastSent: make(map[string]time.Time)}
	ch := &IIOChannel{ID: "ID-ch", Value: 3.0, MinChange: 0.5}

	// the first reading is always published, because nothing has been sent
	if !c.shouldPublish(ch, 3.0, false) {
		t.Error("expected the first reading to publish")
	}

	c.lastSent[ch.ID] = time.Now()

	if c.shouldPublish(ch, 3.1, false) {
		t.Error("expected a move smaller than minChange to be held back")
	}

	if c.shouldPublish(ch, 3.5, false) {
		t.Error("expected a move equal to minChange to be held back")
	}

	if !c.shouldPublish(ch, 4.0, false) {
		t.Error("expected a move past minChange to publish")
	}

	if !c.shouldPublish(ch, 3.1, true) {
		t.Error("expected a forced publish to report an unchanged value")
	}

	// past the refresh interval, an unchanged reading is republished
	c.lastSent[ch.ID] = time.Now().Add(-iioValueRefresh - time.Second)

	if !c.shouldPublish(ch, 3.0, false) {
		t.Error("expected the refresh to republish an unchanged value")
	}
}
