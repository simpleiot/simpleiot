package client

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// IIODevicePath is the root of the Linux IIO sysfs tree. It is a variable so
// that tests can point it at a fixture directory; nothing else should change
// it.
var IIODevicePath = "/sys/bus/iio/devices"

// errIIOAttrMissing reports an attribute the driver does not publish, so that
// an unsupported device setting is distinguished from a failed write.
var errIIOAttrMissing = errors.New("attribute not supported by this device")

// iioDevice is a resolved IIO device directory
type iioDevice struct {
	Path string
	Name string
}

// iioChannelInfo is a channel discovered on a device
type iioChannelInfo struct {
	// Channel is the attribute prefix without _raw, e.g. "in_voltage0"
	Channel string
	// Type is the measured quantity, e.g. "voltage"
	Type string
	// Output is true for an out_* channel
	Output bool
}

// iioUnits are the units published for each channel type. A type that is not
// listed has no unit the ABI fixes, so the value is published as the driver
// reports it and the person configuring the node sets the units.
var iioUnits = map[string]string{
	"voltage":          "V",
	"current":          "A",
	"temp":             "C",
	"accel":            "m/s^2",
	"anglvel":          "rad/s",
	"magn":             "G",
	"pressure":         "kPa",
	"humidityrelative": "%",
	"illuminance":      "lx",
}

// iioMilliTypes are the channel types the ABI reports in milli units. The
// client divides these by a thousand so a temperature compares against 25 and
// a voltage graphs as 3.3, the same as every other client in the system.
var iioMilliTypes = map[string]bool{
	"voltage": true,
	"current": true,
	"temp":    true,
}

// IIOUnits returns the units an IIO channel of this type is published in, or
// an empty string when the ABI does not fix one.
func IIOUnits(channelType string) string {
	return iioUnits[channelType]
}

// iioScaleToBase is the factor the ABI unit is multiplied by to reach the base
// unit this system publishes in
func iioScaleToBase(channelType string) float64 {
	if iioMilliTypes[channelType] {
		return 0.001
	}

	return 1
}

// iioFind resolves a device given a name, an "iio:deviceN" directory name, or
// a path, and reads back its name attribute. Matching by name is tried first,
// because the device number depends on probe order and is not stable across
// boots.
func iioFind(root, device string) (iioDevice, error) {
	if device == "" {
		return iioDevice{}, errors.New("no device set")
	}

	// a full path, or anything else with a separator in it, is used as given
	if strings.ContainsRune(device, filepath.Separator) {
		return iioDeviceAt(device)
	}

	dirs, err := filepath.Glob(filepath.Join(root, "iio:device*"))
	if err != nil {
		return iioDevice{}, err
	}

	sort.Strings(dirs)

	for _, dir := range dirs {
		if filepath.Base(dir) == device {
			return iioDeviceAt(dir)
		}
	}

	for _, dir := range dirs {
		d, err := iioDeviceAt(dir)
		if err != nil {
			continue
		}

		if d.Name == device {
			return d, nil
		}
	}

	return iioDevice{}, fmt.Errorf("IIO device not found: %v", device)
}

// iioDeviceAt reads the device at a directory, which must exist and be
// readable. The name attribute is optional, because a few drivers do not
// publish one.
func iioDeviceAt(path string) (iioDevice, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return iioDevice{}, err
	}

	if !fi.IsDir() {
		return iioDevice{}, fmt.Errorf("not an IIO device directory: %v", path)
	}

	name, err := iioReadAttr(path, "name")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return iioDevice{}, err
	}

	return iioDevice{Path: path, Name: name}, nil
}

// iioChannels lists the channels a device publishes, by globbing the value
// attributes and parsing the prefix off each one. A channel that publishes
// both _raw and _input is listed once.
func iioChannels(dev iioDevice) ([]iioChannelInfo, error) {
	var names []string

	for _, suffix := range []string{"_raw", "_input"} {
		for _, prefix := range []string{"in_", "out_"} {
			matches, err := filepath.Glob(
				filepath.Join(dev.Path, prefix+"*"+suffix))
			if err != nil {
				return nil, err
			}

			for _, m := range matches {
				names = append(names,
					strings.TrimSuffix(filepath.Base(m), suffix))
			}
		}
	}

	sort.Strings(names)

	var chans []iioChannelInfo
	seen := make(map[string]bool)

	for _, n := range names {
		if seen[n] {
			continue
		}
		seen[n] = true

		typ, output, ok := iioParseChannel(n)
		if !ok {
			continue
		}

		chans = append(chans, iioChannelInfo{
			Channel: n,
			Type:    typ,
			Output:  output,
		})
	}

	return chans, nil
}

// iioParseChannel takes an attribute prefix such as "in_voltage0",
// "in_voltage0-voltage1", "in_accel_x", or "out_voltage0" and returns the
// measured quantity and whether the channel is an output. The type is the
// alphabetic run after the direction prefix, stopping at the first digit or
// the differential dash.
func iioParseChannel(name string) (string, bool, bool) {
	var output bool
	var rest string

	switch {
	case strings.HasPrefix(name, "in_"):
		rest = name[len("in_"):]
	case strings.HasPrefix(name, "out_"):
		rest = name[len("out_"):]
		output = true
	default:
		return "", false, false
	}

	end := strings.IndexFunc(rest, func(r rune) bool {
		return r < 'a' || r > 'z'
	})

	if end == 0 {
		return "", false, false
	}

	if end < 0 {
		end = len(rest)
	}

	return rest[:end], output, true
}

// iioRead reads one channel and converts it to the unit this system publishes
// in. A driver that publishes an already converted value in _input is used as
// it stands; otherwise the raw count is read and (raw + offset) * scale is
// applied, with the per channel attribute falling back to the per type one.
func iioRead(dev iioDevice, ch string) (float64, error) {
	typ, _, ok := iioParseChannel(ch)
	if !ok {
		return 0, fmt.Errorf("not an IIO channel name: %v", ch)
	}

	toBase := iioScaleToBase(typ)

	if v, err := iioReadFloat(dev.Path, ch+"_input"); err == nil {
		return v * toBase, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return 0, err
	}

	raw, err := iioReadFloat(dev.Path, ch+"_raw")
	if err != nil {
		return 0, err
	}

	scale, err := iioReadConv(dev, ch, typ, "_scale", 1)
	if err != nil {
		return 0, err
	}

	offset, err := iioReadConv(dev, ch, typ, "_offset", 0)
	if err != nil {
		return 0, err
	}

	return (raw + offset) * scale * toBase, nil
}

// iioWrite converts a value in the published unit back to a raw count and
// writes it to the channel's _raw attribute.
func iioWrite(dev iioDevice, ch string, v float64) error {
	typ, _, ok := iioParseChannel(ch)
	if !ok {
		return fmt.Errorf("not an IIO channel name: %v", ch)
	}

	scale, err := iioReadConv(dev, ch, typ, "_scale", 1)
	if err != nil {
		return err
	}

	if scale == 0 {
		return fmt.Errorf("channel %v has a zero scale", ch)
	}

	offset, err := iioReadConv(dev, ch, typ, "_offset", 0)
	if err != nil {
		return err
	}

	raw := v/(iioScaleToBase(typ)*scale) - offset

	return os.WriteFile(filepath.Join(dev.Path, ch+"_raw"),
		[]byte(strconv.FormatFloat(raw, 'f', -1, 64)), 0644)
}

// iioReadConv reads a conversion attribute, which the ABI allows a driver to
// publish per channel or once for the whole channel type. A driver that
// publishes neither is using the default.
func iioReadConv(dev iioDevice, ch, typ, suffix string, def float64) (float64, error) {
	v, err := iioReadFloat(dev.Path, ch+suffix)
	if err == nil {
		return v, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return 0, err
	}

	// the per type attribute drops the channel index: in_voltage0_scale
	// falls back to in_voltage_scale
	dir := "in_"
	if strings.HasPrefix(ch, "out_") {
		dir = "out_"
	}

	v, err = iioReadFloat(dev.Path, dir+typ+suffix)
	if err == nil {
		return v, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return 0, err
	}

	return def, nil
}

// iioWriteAttr writes a device level attribute such as sampling_frequency. An
// attribute the driver does not publish is reported as errIIOAttrMissing, so
// an unsupported setting is not counted as a failure to write.
func iioWriteAttr(dev iioDevice, attr, v string) error {
	path := filepath.Join(dev.Path, attr)

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%v: %w", attr, errIIOAttrMissing)
		}
		return err
	}

	return os.WriteFile(path, []byte(v), 0644)
}

// iioReadAttr reads one sysfs attribute and trims the newline the kernel adds
func iioReadAttr(dir, attr string) (string, error) {
	d, err := os.ReadFile(filepath.Join(dir, attr))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(d)), nil
}

// iioReadFloat reads a numeric sysfs attribute
func iioReadFloat(dir, attr string) (float64, error) {
	s, err := iioReadAttr(dir, attr)
	if err != nil {
		return 0, err
	}

	if s == "" {
		return 0, fmt.Errorf("no data in %v", filepath.Join(dir, attr))
	}

	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing %v: %w",
			filepath.Join(dir, attr), err)
	}

	return v, nil
}
