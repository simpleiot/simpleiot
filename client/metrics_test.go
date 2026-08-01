package client

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/simpleiot/simpleiot/data"
)

// writeSysfs creates a sysfs style directory of attribute files. An empty value
// means the attribute is left out, which is how an unreadable one appears.
func writeSysfs(t *testing.T, dir, node string, attrs map[string]string) {
	t.Helper()

	n := filepath.Join(dir, node)
	if err := os.MkdirAll(n, 0755); err != nil {
		t.Fatal("Error creating sysfs dir:", err)
	}

	for name, val := range attrs {
		if val == "" {
			continue
		}

		if err := os.WriteFile(filepath.Join(n, name), []byte(val+"\n"), 0644); err != nil {
			t.Fatal("Error writing sysfs attribute:", err)
		}
	}
}

// checkReadings compares readings against what the fixture should produce
func checkReadings(t *testing.T, what string, got, exp []reading) {
	t.Helper()

	if len(got) != len(exp) {
		t.Fatalf("Expected %v %v readings, got %v: %+v", len(exp), what, len(got), got)
	}

	for i, e := range exp {
		if got[i] != e {
			t.Errorf("Expected %v reading %v to be %+v, got %+v", what, i, e, got[i])
		}
	}
}

func TestThermalZones(t *testing.T) {
	dir := t.TempDir()

	writeSysfs(t, dir, "thermal_zone0", map[string]string{"type": "cpu-thermal", "temp": "37718"})
	// a powered down rail returns an error when its temp is read
	writeSysfs(t, dir, "thermal_zone1", map[string]string{"type": "gpu-thermal"})
	// a zone that reports something we cannot parse
	writeSysfs(t, dir, "thermal_zone2", map[string]string{"type": "cv0-thermal", "temp": "unavailable"})
	writeSysfs(t, dir, "thermal_zone3", map[string]string{"type": "tj-thermal", "temp": "37781"})
	// unrelated entries in the same directory are left alone
	writeSysfs(t, dir, "cooling_device0", map[string]string{"type": "pwm-fan", "cur_state": "1"})

	checkReadings(t, "zone", thermalZones(dir), []reading{
		{key: "cpu-thermal", val: 37.718},
		{key: "tj-thermal", val: 37.781},
	})
}

func TestThermalZonesNoInterface(t *testing.T) {
	if temps := thermalZones(filepath.Join(t.TempDir(), "does-not-exist")); len(temps) != 0 {
		t.Error("Expected no readings without a thermal interface, got:", temps)
	}
}

func TestCoolingDevices(t *testing.T) {
	dir := t.TempDir()

	writeSysfs(t, dir, "cooling_device0", map[string]string{
		"type": "cpufreq-cpu0", "cur_state": "0", "max_state": "28",
	})
	// a device throttling the system, which is what we are after
	writeSysfs(t, dir, "cooling_device1", map[string]string{
		"type": "devfreq-17000000.gpu", "cur_state": "4", "max_state": "10",
	})
	// a device that reports no state is skipped for that reading alone
	writeSysfs(t, dir, "cooling_device2", map[string]string{
		"type": "hot-surface-alert", "max_state": "2",
	})
	// a device with no type at all cannot be published
	writeSysfs(t, dir, "cooling_device3", map[string]string{"cur_state": "1"})
	writeSysfs(t, dir, "thermal_zone0", map[string]string{"type": "cpu-thermal", "temp": "37718"})

	state, stateMax := coolingDevices(dir)

	checkReadings(t, "cooling state", state, []reading{
		{key: "cpufreq-cpu0", val: 0},
		{key: "devfreq-17000000.gpu", val: 4},
	})

	checkReadings(t, "cooling max", stateMax, []reading{
		{key: "cpufreq-cpu0", val: 28},
		{key: "devfreq-17000000.gpu", val: 10},
		{key: "hot-surface-alert", val: 2},
	})
}

func TestFans(t *testing.T) {
	dir := t.TempDir()

	// the standard hwmon layout
	writeSysfs(t, dir, "hwmon0", map[string]string{
		"name": "nct6779", "fan1_input": "1200", "fan2_input": "900",
		"pwm1": "128", "pwm1_enable": "1", "pwm1_mode": "1",
	})
	// the Tegra tachometer, which reports rpm rather than fan1_input
	writeSysfs(t, dir, "hwmon1", map[string]string{"name": "pwm_tach", "rpm": "717"})
	writeSysfs(t, dir, "hwmon2", map[string]string{"name": "pwmfan", "pwm1": "54"})
	// a temperature sensor has neither, and is no trouble
	writeSysfs(t, dir, "hwmon3", map[string]string{"name": "tmp451", "temp1_input": "27187"})

	rpm, pwm := fans(dir)

	checkReadings(t, "fan speed", rpm, []reading{
		{key: "nct6779", val: 1200},
		{key: "nct6779", val: 900},
		{key: "pwm_tach", val: 717},
	})

	// pwm1_enable and pwm1_mode are settings, not drive levels
	checkReadings(t, "fan pwm", pwm, []reading{
		{key: "nct6779", val: 128},
		{key: "pwmfan", val: 54},
	})
}

func TestPowerRails(t *testing.T) {
	dir := t.TempDir()

	// an INA3221, whose channel 3 is disabled and so cannot be read. in4 and
	// in5 are the shunt voltages the driver exposes without labels.
	writeSysfs(t, dir, "hwmon0", map[string]string{
		"name": "ina3221",
		"in1_label": "VDD_GPU_SOC", "in1_input": "18792", "curr1_input": "140",
		"in2_label": "VDD_CPU_CV", "in2_input": "18784", "curr2_input": "20",
		"in3_label": "VIN_SYS_5V0",
		"in4_input": "280", "in5_input": "40",
		"in7_label": "sum of shunt voltages", "in7_input": "360",
	})
	// a driver that reports power itself, in microwatts
	writeSysfs(t, dir, "hwmon1", map[string]string{
		"name": "ina226", "in1_label": "VDD_5V", "in1_input": "5016",
		"curr1_input": "820", "power1_input": "4113120",
	})
	// no labels means nothing to name a rail with
	writeSysfs(t, dir, "hwmon2", map[string]string{"name": "pwmfan", "pwm1": "54"})

	volts, amps, watts := powerRails(dir)

	checkReadings(t, "voltage", volts, []reading{
		{key: "VDD_GPU_SOC", val: 18.792},
		{key: "VDD_CPU_CV", val: 18.784},
		{key: "sum_of_shunt_voltages", val: 0.36},
		{key: "VDD_5V", val: 5.016},
	})

	checkReadings(t, "current", amps, []reading{
		{key: "VDD_GPU_SOC", val: 0.14},
		{key: "VDD_CPU_CV", val: 0.02},
		{key: "VDD_5V", val: 0.82},
	})

	// the first two are the product of voltage and current, the last is what
	// the driver reported
	checkReadings(t, "power", watts, []reading{
		{key: "VDD_GPU_SOC", val: 18.792 * 0.14},
		{key: "VDD_CPU_CV", val: 18.784 * 0.02},
		{key: "VDD_5V", val: 4.11312},
	})
}

func TestCPUFreqs(t *testing.T) {
	dir := t.TempDir()

	writeSysfs(t, dir, filepath.Join("cpu0", "cpufreq"), map[string]string{"scaling_cur_freq": "1420800"})
	writeSysfs(t, dir, filepath.Join("cpu1", "cpufreq"), map[string]string{"scaling_cur_freq": "729600"})
	// an offline core cannot be read
	writeSysfs(t, dir, "cpu2", map[string]string{"online": "0"})
	// the sibling directories in this part of sysfs are not CPUs
	writeSysfs(t, dir, "cpufreq", map[string]string{"scaling_cur_freq": "1"})
	writeSysfs(t, dir, "cpuidle", map[string]string{"scaling_cur_freq": "1"})

	checkReadings(t, "cpu freq", cpuFreqs(dir), []reading{
		{key: "cpu0", val: 1420.8},
		{key: "cpu1", val: 729.6},
	})
}

func TestReadingPointsUniqueKeys(t *testing.T) {
	// sensor names are not unique, and duplicate keys would overwrite each
	// other in the point store
	pts := readingPoints(data.PointTypeMetricSysFanSpeed, []reading{
		{key: "nct6779", val: 1200},
		{key: "nct6779", val: 900},
		{key: "pwm_tach", val: 717},
		// a reading we cannot name is dropped
		{key: "", val: 1},
	})

	exp := []struct {
		key string
		val float64
	}{
		{"nct6779", 1200},
		{"nct6779_2", 900},
		{"pwm_tach", 717},
	}

	if len(pts) != len(exp) {
		t.Fatalf("Expected %v points, got %v: %v", len(exp), len(pts), pts)
	}

	for i, e := range exp {
		if pts[i].Type != data.PointTypeMetricSysFanSpeed {
			t.Error("Expected fan speed points, got:", pts[i].Type)
		}

		if pts[i].Key != e.key {
			t.Errorf("Expected point %v key to be %v, got %v", i, e.key, pts[i].Key)
		}

		if pts[i].Val() != e.val {
			t.Errorf("Expected %v to be %v, got %v", e.key, e.val, pts[i].Val())
		}
	}
}

func TestSysTemperaturesIncludesZones(t *testing.T) {
	dir := t.TempDir()

	writeSysfs(t, dir, "thermal_zone0", map[string]string{"type": "tj-thermal", "temp": "37781"})

	old := thermalPath
	thermalPath = dir
	defer func() { thermalPath = old }()

	var found bool

	for _, p := range sysTemperatures() {
		if p.Type != data.PointTypeTemperature {
			t.Error("Expected temperature points, got:", p.Type)
		}

		if p.Key == "tj-thermal" {
			found = true

			if p.Val() != 37.781 {
				t.Error("Expected tj-thermal to be 37.781, got:", p.Val())
			}
		}
	}

	if !found {
		t.Error("Expected the thermal zone reading to be collected")
	}
}
