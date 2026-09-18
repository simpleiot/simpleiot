package client

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeName(t *testing.T) {
	for _, test := range []struct {
		name string
		ok   bool
	}{
		{"in_voltage0", true},
		{"28-000005e2fdc3", true},
		{"alarm.wav", true},
		{"", false},
		{".", false},
		{"..", false},
		{"../etc/passwd", false},
		{"a/b", false},
		{`a\b`, false},
		{"a\x00b", false},
	} {
		err := safeName(test.name)
		if test.ok && err != nil {
			t.Errorf("%q: unexpected error %v", test.name, err)
		}
		if !test.ok && err == nil {
			t.Errorf("%q: expected error", test.name)
		}
	}
}

func TestSafePath(t *testing.T) {
	for _, test := range []struct {
		path string
		ok   bool
	}{
		{"lsm6dsl", true},
		{"iio:device0", true},
		{"/sys/bus/iio/devices/iio:device0", true},
		{"/data/alarm.wav", true},
		{"", false},
		{"..", false},
		{"../x", false},
		{"data/alarm.wav", false},
		{"/sys/bus/iio/devices/../../../etc", false},
		{"/data//alarm.wav", false},
		{"/data/./alarm.wav", false},
		{"/data/alarm.wav/", false},
		{"/data/\x00", false},
	} {
		err := safePath(test.path)
		if test.ok && err != nil {
			t.Errorf("%q: unexpected error %v", test.path, err)
		}
		if !test.ok && err == nil {
			t.Errorf("%q: expected error", test.path)
		}
	}
}

// TestIIOPathsFromPoints checks that a device or channel taken from a point
// cannot reach outside the sysfs directory.
func TestIIOPathsFromPoints(t *testing.T) {
	root := t.TempDir()
	path0 := writeIIOFixture(t, root, "iio:device0", map[string]string{
		"name":            "ads1015",
		"in_voltage0_raw": "100",
	})

	// a sibling directory that must not be reachable
	outside := filepath.Join(root, "..", filepath.Base(root)+"-outside")
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(outside) })
	if err := os.WriteFile(filepath.Join(outside, "name"), []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := iioFind(root, root+"/../"+filepath.Base(outside)); err == nil {
		t.Error("iioFind followed .. in a device path")
	}
	if _, err := iioFind(root, "../"+filepath.Base(outside)); err == nil {
		t.Error("iioFind accepted a relative path")
	}
	if dev, err := iioFind(root, path0); err != nil || dev.Name != "ads1015" {
		t.Errorf("iioFind refused a clean absolute path: %v %v", dev, err)
	}

	dev, err := iioFind(root, "ads1015")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := iioRead(dev, "in_voltage0"); err != nil {
		t.Errorf("iioRead refused a channel: %v", err)
	}
	if _, err := iioRead(dev, "in_voltage0/../../"+filepath.Base(outside)+"/name"); err == nil {
		t.Error("iioRead followed .. in a channel")
	}
	if err := iioWrite(dev, "out_voltage0/../x", 1); err == nil {
		t.Error("iioWrite followed .. in a channel")
	}
}

// TestOneWirePathFromPoint checks that a sensor ID taken from a point cannot
// reach outside the device directory.
func TestOneWirePathFromPoint(t *testing.T) {
	root := t.TempDir()
	writeW1(t, root, 0, "28-celsius", "23456")

	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "temperature"), []byte("1000"), 0644); err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(root, outside)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := oneWireRead(root, rel, ""); err == nil {
		t.Error("oneWireRead followed .. in a device ID")
	}
	if _, err := oneWireRead(root, outside, ""); err == nil {
		t.Error("oneWireRead accepted an absolute path as a device ID")
	}
	if _, err := oneWireRead(root, "28-celsius", ""); err != nil {
		t.Errorf("oneWireRead refused a sensor ID: %v", err)
	}
}
