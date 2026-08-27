package client_test

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

const (
	// iioTestTimeout is how long a case waits for a point to reach a node
	iioTestTimeout = time.Second * 3

	// iioTestPoll is the poll period the cases run at, fast enough to keep
	// the suite quick and slow enough to leave the store time to settle
	iioTestPoll = 50

	// iioTestIdle is a poll period long enough that the client does not tick
	// while a case is still building its nodes
	iioTestIdle = 600000
)

// iioTest is a SIOT test server pointed at a fixture directory laid out like
// the IIO sysfs tree, so the cases exercise the real read path with no
// hardware involved.
type iioTest struct {
	t      *testing.T
	nc     *nats.Conn
	root   data.NodeEdge
	fsRoot string
	// stops holds the node watchers to shut down when the test ends
	stops []func()
}

func newIIOTest(t *testing.T) *iioTest {
	fsRoot := t.TempDir()

	saved := client.IIODevicePath
	client.IIODevicePath = fsRoot
	t.Cleanup(func() { client.IIODevicePath = saved })

	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	g := &iioTest{t: t, nc: nc, root: root, fsRoot: fsRoot}

	t.Cleanup(func() {
		for _, s := range g.stops {
			s()
		}
		stop()
	})

	return g
}

// fixture lays out one device directory from a map of attribute names to
// contents, and returns the directory name to set as the node's device
func (g *iioTest) fixture(dir string, attrs map[string]string) string {
	g.t.Helper()

	path := filepath.Join(g.fsRoot, dir)
	if err := os.MkdirAll(path, 0755); err != nil {
		g.t.Fatal("Error creating IIO device dir:", err)
	}

	for name, content := range attrs {
		g.write(dir, name, content)
	}

	return dir
}

// write sets one attribute, which is how a case moves a reading
func (g *iioTest) write(dir, attr, content string) {
	g.t.Helper()

	err := os.WriteFile(filepath.Join(g.fsRoot, dir, attr),
		[]byte(content+"\n"), 0644)
	if err != nil {
		g.t.Fatalf("Error writing IIO attribute %v: %v", attr, err)
	}
}

// read returns the contents of an attribute, which is how a case checks what
// the client wrote
func (g *iioTest) read(dir, attr string) string {
	g.t.Helper()

	d, err := os.ReadFile(filepath.Join(g.fsRoot, dir, attr))
	if err != nil {
		g.t.Fatalf("Error reading IIO attribute %v: %v", attr, err)
	}

	return strings.TrimSpace(string(d))
}

// remove deletes an attribute, which is how a case makes a read fail
func (g *iioTest) remove(dir, attr string) {
	g.t.Helper()

	if err := os.Remove(filepath.Join(g.fsRoot, dir, attr)); err != nil {
		g.t.Fatalf("Error removing IIO attribute %v: %v", attr, err)
	}
}

// addDevice creates an iio node and returns a getter for its current state
func (g *iioTest) addDevice(cfg client.IIO) func() client.IIO {
	g.t.Helper()

	cfg.Parent = g.root.ID

	if err := client.SendNodeType(g.nc, cfg, "test"); err != nil {
		g.t.Fatalf("Error sending %v node: %v", cfg.Description, err)
	}

	get, stop, err := client.NodeWatcher[client.IIO](g.nc, cfg.ID, cfg.Parent)
	if err != nil {
		g.t.Fatalf("Error setting up watcher for %v: %v", cfg.Description, err)
	}

	g.stops = append(g.stops, stop)

	return get
}

// addChannel creates an iioChannel node under a device and returns a getter
// for its current state. A channel the client would otherwise detect is added
// by hand when a case needs to know its node ID up front.
func (g *iioTest) addChannel(cfg client.IIOChannel) func() client.IIOChannel {
	g.t.Helper()

	if err := client.SendNodeType(g.nc, cfg, "test"); err != nil {
		g.t.Fatalf("Error sending %v channel: %v", cfg.Channel, err)
	}

	get, stop, err := client.NodeWatcher[client.IIOChannel](g.nc, cfg.ID, cfg.Parent)
	if err != nil {
		g.t.Fatalf("Error setting up watcher for %v: %v", cfg.Channel, err)
	}

	g.stops = append(g.stops, stop)

	return get
}

// channels lists the channel nodes under a device, which is how the detection
// cases see what the client created
func (g *iioTest) channels(devID string) []client.IIOChannel {
	g.t.Helper()

	nodes, err := client.GetNodes(g.nc, devID, "all", data.NodeTypeIIOChannel, false)
	if err != nil {
		g.t.Fatal("Error getting channel nodes:", err)
	}

	chans := make([]client.IIOChannel, len(nodes))

	for i, n := range nodes {
		err := data.Decode(data.NodeEdgeChildren{NodeEdge: n}, &chans[i])
		if err != nil {
			g.t.Fatal("Error decoding channel node:", err)
		}
	}

	return chans
}

// start sets the poll period, which is how a case begins polling once its
// nodes are in place. A device is created with a long period so that detection
// does not race the channel nodes the case adds by hand.
func (g *iioTest) start(devID string) {
	g.t.Helper()
	g.send(devID, data.NewPointFloat(data.PointTypePollPeriod, "", iioTestPoll))
}

// send publishes points on a node as if they came from the UI or an upstream
// instance. The origin is set so the client manager passes them to the client.
func (g *iioTest) send(id string, pts ...data.Point) {
	g.t.Helper()

	for i := range pts {
		pts[i].Origin = "test"
	}

	if err := client.SendNodePoints(g.nc, id, pts, true); err != nil {
		g.t.Fatalf("Error sending points to %v: %v", id, err)
	}
}

// iioWait blocks until the node satisfies cond, and fails the test with the
// node's current state when it does not
func iioWait[T any](t *testing.T, get func() T, timeout time.Duration,
	msg string, cond func(T) bool) T {
	t.Helper()

	start := time.Now()

	for {
		cur := get()
		if cond(cur) {
			return cur
		}

		if time.Since(start) > timeout {
			t.Fatalf("Timeout waiting for %v, node is: %+v", msg, cur)
		}

		time.Sleep(time.Millisecond * 10)
	}
}

// iioNear reports whether a published value matches what the conversion chain
// should have produced
func iioNear(got, want float64) bool {
	return math.Abs(got-want) < 1e-6
}

// TestIIODevice covers resolving the device the node names and the device
// level settings that apply to every channel on it.
func TestIIODevice(t *testing.T) {
	g := newIIOTest(t)

	t.Run("resolve by name", func(t *testing.T) {
		g.fixture("iio:device0", map[string]string{"name": "ads1015"})
		g.fixture("iio:device1", map[string]string{"name": "lsm6dsl"})

		dev := g.addDevice(client.IIO{
			ID:          "ID-iio-by-name",
			Description: "resolve by name",
			Device:      "lsm6dsl",
			PollPeriod:  iioTestPoll,
		})

		cur := iioWait(t, dev, iioTestTimeout, "the device to be resolved",
			func(c client.IIO) bool { return c.Connected })

		if filepath.Base(cur.DevicePath) != "iio:device1" {
			t.Errorf("Expected iio:device1, got %v", cur.DevicePath)
		}

		if cur.DeviceName != "lsm6dsl" {
			t.Errorf("Expected the device name lsm6dsl, got %v", cur.DeviceName)
		}
	})

	t.Run("resolve by directory", func(t *testing.T) {
		g.fixture("iio:device2", map[string]string{"name": "ads1115"})

		dev := g.addDevice(client.IIO{
			ID:          "ID-iio-by-dir",
			Description: "resolve by directory",
			Device:      "iio:device2",
			PollPeriod:  iioTestPoll,
		})

		iioWait(t, dev, iioTestTimeout, "the name attribute to be published",
			func(c client.IIO) bool {
				return c.Connected && c.DeviceName == "ads1115" &&
					filepath.Base(c.DevicePath) == "iio:device2"
			})
	})

	t.Run("missing device", func(t *testing.T) {
		dev := g.addDevice(client.IIO{
			ID:          "ID-iio-missing",
			Description: "no such device",
			Device:      "mcp3008",
			PollPeriod:  iioTestPoll,
		})

		// a second failure shows the client keeps trying, which is what
		// covers a driver that has not probed yet
		iioWait(t, dev, iioTestTimeout, "the device to be reported missing",
			func(c client.IIO) bool {
				return !c.Connected && c.Error != "" && c.ErrorCount >= 2
			})

		// naming a device that is there recovers without touching the node
		// again
		g.fixture("iio:device3", map[string]string{"name": "mcp3008"})

		iioWait(t, dev, iioTestTimeout, "the device to be found once it appears",
			func(c client.IIO) bool { return c.Connected && c.Error == "" })
	})

	t.Run("device attributes", func(t *testing.T) {
		dir := g.fixture("iio:device4", map[string]string{
			"name":               "ads1015-settings",
			"sampling_frequency": "1600",
			"in_voltage0_raw":    "1583",
		})

		dev := g.addDevice(client.IIO{
			ID:              "ID-iio-attrs",
			Description:     "device settings",
			Device:          "ads1015-settings",
			PollPeriod:      iioTestPoll,
			SampleFrequency: 100,
		})

		iioWait(t, dev, iioTestTimeout, "the device to connect",
			func(c client.IIO) bool { return c.Connected })

		iioWait(t, func() string { return g.read(dir, "sampling_frequency") },
			iioTestTimeout, "the sample frequency to be written",
			func(s string) bool { return s == "100" })

		// changing the setting on the node writes it through
		g.send("ID-iio-attrs",
			data.NewPointFloat(data.PointTypeSampleFrequency, "", 128))

		iioWait(t, func() string { return g.read(dir, "sampling_frequency") },
			iioTestTimeout, "the new sample frequency to be written",
			func(s string) bool { return s == "128" })
	})

	t.Run("unsupported attribute", func(t *testing.T) {
		// this device publishes no oversampling_ratio, which is not a fault
		g.fixture("iio:device5", map[string]string{
			"name":            "ads1015-no-oversampling",
			"in_voltage0_raw": "1583",
		})

		dev := g.addDevice(client.IIO{
			ID:           "ID-iio-unsupported",
			Description:  "unsupported setting",
			Device:       "ads1015-no-oversampling",
			PollPeriod:   iioTestPoll,
			Oversampling: 4,
		})

		iioWait(t, dev, iioTestTimeout, "the device to connect",
			func(c client.IIO) bool { return c.Connected })

		// give the client a few polls to prove nothing is counted against it
		time.Sleep(time.Millisecond * 200)

		cur := dev()
		if cur.ErrorCount != 0 || cur.Error != "" {
			t.Errorf("An unsupported setting was counted as an error, node is: %+v", cur)
		}
	})
}

// TestIIODetect covers turning the channels a device publishes into nodes.
func TestIIODetect(t *testing.T) {
	g := newIIOTest(t)

	g.fixture("iio:device0", map[string]string{
		"name":              "ads1015",
		"in_voltage0_raw":   "1583",
		"in_voltage0_scale": "2.0",
		"in_voltage1_raw":   "1584",
		"in_temp_raw":       "2634",
		"in_temp_scale":     "0.062500000",
		"in_temp_offset":    "-1092",
		"out_voltage0_raw":  "2048",
	})

	dev := g.addDevice(client.IIO{
		ID:          "ID-iio-detect",
		Description: "detected channels",
		Device:      "ads1015",
		PollPeriod:  iioTestPoll,
	})

	iioWait(t, dev, iioTestTimeout, "the device to connect",
		func(c client.IIO) bool { return c.Connected })

	chans := iioWait(t, func() []client.IIOChannel { return g.channels("ID-iio-detect") },
		iioTestTimeout, "every channel to be added",
		func(c []client.IIOChannel) bool { return len(c) == 4 })

	want := map[string]struct {
		typ       string
		direction string
		units     string
	}{
		"in_temp":      {"temp", data.PointValueInput, "C"},
		"in_voltage0":  {"voltage", data.PointValueInput, "V"},
		"in_voltage1":  {"voltage", data.PointValueInput, "V"},
		"out_voltage0": {"voltage", data.PointValueOutput, "V"},
	}

	for _, ch := range chans {
		w, ok := want[ch.Channel]
		if !ok {
			t.Fatalf("Unexpected channel %v", ch.Channel)
		}

		if ch.ChannelType != w.typ {
			t.Errorf("%v: expected type %v, got %v", ch.Channel, w.typ, ch.ChannelType)
		}

		if ch.Direction != w.direction {
			t.Errorf("%v: expected direction %v, got %v", ch.Channel, w.direction, ch.Direction)
		}

		if ch.Units != w.units {
			t.Errorf("%v: expected units %v, got %v", ch.Channel, w.units, ch.Units)
		}
	}

	// several more polls must not add the same channel twice
	time.Sleep(time.Millisecond * 300)

	if again := g.channels("ID-iio-detect"); len(again) != 4 {
		t.Fatalf("Expected 4 channels after more polls, got %v", len(again))
	}
}

// TestIIORead covers the conversion chain a reading passes through and what
// decides whether it is published.
func TestIIOChannelRead(t *testing.T) {
	g := newIIOTest(t)

	t.Run("conversion", func(t *testing.T) {
		g.fixture("iio:device0", map[string]string{
			"name":              "ads1015",
			"in_voltage0_raw":   "1583",
			"in_voltage0_scale": "2.0",
			"in_temp_raw":       "2634",
			"in_temp_scale":     "0.062500000",
			"in_temp_offset":    "-1092",
		})

		dev := g.addDevice(client.IIO{
			ID:          "ID-iio-convert",
			Description: "conversion",
			Device:      "ads1015",
			PollPeriod:  iioTestPoll,
		})

		iioWait(t, dev, iioTestTimeout, "the device to connect",
			func(c client.IIO) bool { return c.Connected })

		values := iioWait(t, func() []client.IIOChannel { return g.channels("ID-iio-convert") },
			iioTestTimeout, "both channels to be detected and read",
			func(cs []client.IIOChannel) bool {
				if len(cs) != 2 {
					return false
				}

				for _, c := range cs {
					if c.Value == 0 {
						return false
					}
				}

				return true
			})

		for _, ch := range values {
			switch ch.Channel {
			case "in_voltage0":
				// 1583 counts at 2 mV per count is 3166 mV, published in volts
				if !iioNear(ch.Value, 3.166) {
					t.Errorf("Expected 3.166 V, got %v", ch.Value)
				}
			case "in_temp":
				// the offset is applied before the scale, and the result is
				// millidegrees
				if !iioNear(ch.Value, (2634-1092)*0.0625/1000) {
					t.Errorf("Expected %v C, got %v", (2634-1092)*0.0625/1000, ch.Value)
				}
			}
		}
	})

	t.Run("node scale and offset", func(t *testing.T) {
		// a 4-20 mA loop across a 100 ohm sense resistor, read as a current
		// channel: 4 mA is 0% and 20 mA is 100%
		dir := g.fixture("iio:device1", map[string]string{
			"name":              "loop",
			"in_current0_raw":   "4",
			"in_current0_scale": "1.0",
		})

		dev := g.addDevice(client.IIO{
			ID:          "ID-iio-eng",
			Description: "engineering units",
			Device:      "loop",
			PollPeriod:  iioTestIdle,
		})

		ch := g.addChannel(client.IIOChannel{
			ID:          "ID-iio-eng-ch",
			Parent:      "ID-iio-eng",
			Description: "tank level",
			Channel:     "in_current0",
			ChannelType: "current",
			Direction:   data.PointValueInput,
			Units:       "%",
			// the ABI reports milliamps, which the client publishes in amps,
			// so the node converts amps to percent
			Scale:  6250,
			Offset: -25,
		})

		g.start("ID-iio-eng")

		iioWait(t, dev, iioTestTimeout, "the device to connect",
			func(c client.IIO) bool { return c.Connected })

		// 4 mA is 0.004 A, which is 0 percent
		iioWait(t, ch, iioTestTimeout, "the bottom of the loop range",
			func(c client.IIOChannel) bool { return iioNear(c.Value, 0) })

		g.write(dir, "in_current0_raw", "20")

		// 20 mA is 0.02 A, which is 100 percent
		iioWait(t, ch, iioTestTimeout, "the top of the loop range",
			func(c client.IIOChannel) bool { return iioNear(c.Value, 100) })
	})

	t.Run("min change", func(t *testing.T) {
		dir := g.fixture("iio:device2", map[string]string{
			"name":              "noisy",
			"in_voltage0_raw":   "1000",
			"in_voltage0_scale": "1.0",
		})

		dev := g.addDevice(client.IIO{
			ID:          "ID-iio-minchange",
			Description: "min change",
			Device:      "noisy",
			PollPeriod:  iioTestIdle,
		})

		ch := g.addChannel(client.IIOChannel{
			ID:          "ID-iio-minchange-ch",
			Parent:      "ID-iio-minchange",
			Description: "noisy channel",
			Channel:     "in_voltage0",
			ChannelType: "voltage",
			Direction:   data.PointValueInput,
			Scale:       1,
			MinChange:   0.5,
		})

		g.start("ID-iio-minchange")

		iioWait(t, dev, iioTestTimeout, "the device to connect",
			func(c client.IIO) bool { return c.Connected })

		iioWait(t, ch, iioTestTimeout, "the first reading",
			func(c client.IIOChannel) bool { return iioNear(c.Value, 1.0) })

		// a move smaller than minChange is held back
		g.write(dir, "in_voltage0_raw", "1100")
		time.Sleep(time.Millisecond * 300)

		if cur := ch(); !iioNear(cur.Value, 1.0) {
			t.Errorf("A move smaller than minChange was published, node is: %+v", cur)
		}

		// a move past it is published, and reports where the reading actually
		// is rather than where it was when it was last sent
		g.write(dir, "in_voltage0_raw", "2000")

		iioWait(t, ch, iioTestTimeout, "the move past minChange",
			func(c client.IIOChannel) bool { return iioNear(c.Value, 2.0) })
	})

	t.Run("read failure", func(t *testing.T) {
		dir := g.fixture("iio:device3", map[string]string{
			"name":              "failing",
			"in_voltage0_raw":   "1000",
			"in_voltage0_scale": "1.0",
			"in_voltage1_raw":   "2000",
			"in_voltage1_scale": "1.0",
		})

		dev := g.addDevice(client.IIO{
			ID:          "ID-iio-fail",
			Description: "read failure",
			Device:      "failing",
			PollPeriod:  iioTestIdle,
		})

		bad := g.addChannel(client.IIOChannel{
			ID:          "ID-iio-fail-bad",
			Parent:      "ID-iio-fail",
			Description: "channel that stops reading",
			Channel:     "in_voltage0",
			ChannelType: "voltage",
			Direction:   data.PointValueInput,
			Scale:       1,
		})

		good := g.addChannel(client.IIOChannel{
			ID:          "ID-iio-fail-good",
			Parent:      "ID-iio-fail",
			Description: "channel that keeps reading",
			Channel:     "in_voltage1",
			ChannelType: "voltage",
			Direction:   data.PointValueInput,
			Scale:       1,
		})

		g.start("ID-iio-fail")

		iioWait(t, bad, iioTestTimeout, "the first reading",
			func(c client.IIOChannel) bool { return iioNear(c.Value, 1.0) })

		g.remove(dir, "in_voltage0_raw")

		iioWait(t, bad, iioTestTimeout, "the failed read to be reported",
			func(c client.IIOChannel) bool {
				return c.Error != "" && c.ErrorCount >= 1
			})

		iioWait(t, dev, iioTestTimeout, "the device to count the failed read",
			func(c client.IIO) bool { return c.ErrorCount >= 1 })

		// the other channel is unaffected
		g.write(dir, "in_voltage1_raw", "3000")

		iioWait(t, good, iioTestTimeout, "the other channel to keep reading",
			func(c client.IIOChannel) bool {
				return iioNear(c.Value, 3.0) && c.Error == ""
			})
	})

	t.Run("disabled channel", func(t *testing.T) {
		dir := g.fixture("iio:device4", map[string]string{
			"name":              "partly-disabled",
			"in_voltage0_raw":   "1000",
			"in_voltage0_scale": "1.0",
			"in_voltage1_raw":   "2000",
			"in_voltage1_scale": "1.0",
		})

		g.addDevice(client.IIO{
			ID:          "ID-iio-disable",
			Description: "disabled channel",
			Device:      "partly-disabled",
			PollPeriod:  iioTestIdle,
		})

		off := g.addChannel(client.IIOChannel{
			ID:          "ID-iio-disable-off",
			Parent:      "ID-iio-disable",
			Description: "disabled channel",
			Channel:     "in_voltage0",
			ChannelType: "voltage",
			Direction:   data.PointValueInput,
			Scale:       1,
			Disabled:    true,
		})

		on := g.addChannel(client.IIOChannel{
			ID:          "ID-iio-disable-on",
			Parent:      "ID-iio-disable",
			Description: "enabled channel",
			Channel:     "in_voltage1",
			ChannelType: "voltage",
			Direction:   data.PointValueInput,
			Scale:       1,
		})

		g.start("ID-iio-disable")

		iioWait(t, on, iioTestTimeout, "the enabled channel to be read",
			func(c client.IIOChannel) bool { return iioNear(c.Value, 2.0) })

		g.write(dir, "in_voltage0_raw", "5000")
		g.write(dir, "in_voltage1_raw", "6000")

		iioWait(t, on, iioTestTimeout, "the enabled channel to keep up",
			func(c client.IIOChannel) bool { return iioNear(c.Value, 6.0) })

		if cur := off(); cur.Value != 0 {
			t.Errorf("A disabled channel was read, node is: %+v", cur)
		}
	})
}

// TestIIOWrite covers driving an output channel and what happens when a value
// is written to an input.
func TestIIOChannelWrite(t *testing.T) {
	g := newIIOTest(t)

	t.Run("output write", func(t *testing.T) {
		dir := g.fixture("iio:device0", map[string]string{
			"name":               "dac",
			"out_voltage0_raw":   "0",
			"out_voltage0_scale": "2.0",
		})

		dev := g.addDevice(client.IIO{
			ID:          "ID-iio-out",
			Description: "output",
			Device:      "dac",
			PollPeriod:  iioTestIdle,
		})

		ch := g.addChannel(client.IIOChannel{
			ID:          "ID-iio-out-ch",
			Parent:      "ID-iio-out",
			Description: "analog out",
			Channel:     "out_voltage0",
			ChannelType: "voltage",
			Direction:   data.PointValueOutput,
			Units:       "V",
			Scale:       1,
		})

		g.start("ID-iio-out")

		iioWait(t, dev, iioTestTimeout, "the device to connect",
			func(c client.IIO) bool { return c.Connected })

		g.send("ID-iio-out-ch",
			data.NewPointFloat(data.PointTypeValueSet, "", 1.5))

		// 1.5 V is 1500 mV, which at 2 mV per count is a count of 750
		iioWait(t, func() string { return g.read(dir, "out_voltage0_raw") },
			iioTestTimeout, "the raw count to be written",
			func(s string) bool { return s == "750" })

		iioWait(t, ch, iioTestTimeout, "the value read back from the channel",
			func(c client.IIOChannel) bool {
				return iioNear(c.Value, 1.5) && c.Error == ""
			})
	})

	t.Run("write to an input", func(t *testing.T) {
		dir := g.fixture("iio:device1", map[string]string{
			"name":              "adc",
			"in_voltage0_raw":   "1000",
			"in_voltage0_scale": "1.0",
		})

		g.addDevice(client.IIO{
			ID:          "ID-iio-in",
			Description: "input written to",
			Device:      "adc",
			PollPeriod:  iioTestIdle,
		})

		ch := g.addChannel(client.IIOChannel{
			ID:          "ID-iio-in-ch",
			Parent:      "ID-iio-in",
			Description: "analog in",
			Channel:     "in_voltage0",
			ChannelType: "voltage",
			Direction:   data.PointValueInput,
			Units:       "V",
			Scale:       1,
		})

		g.start("ID-iio-in")

		iioWait(t, ch, iioTestTimeout, "the first reading",
			func(c client.IIOChannel) bool { return iioNear(c.Value, 1.0) })

		g.send("ID-iio-in-ch",
			data.NewPointFloat(data.PointTypeValueSet, "", 2.5))

		iioWait(t, ch, iioTestTimeout, "the write to be reported as an error",
			func(c client.IIOChannel) bool {
				return c.Error != "" && c.ErrorCount >= 1
			})

		if got := g.read(dir, "in_voltage0_raw"); got != "1000" {
			t.Errorf("An input channel was written to, raw is now %v", got)
		}
	})
}

// TestIIORule wires the loop someone would actually build: an input channel
// drives a rule condition, and the rule action writes an output channel.
func TestIIORule(t *testing.T) {
	g := newIIOTest(t)

	dir := g.fixture("iio:device0", map[string]string{
		"name":               "loop-controller",
		"in_voltage0_raw":    "1000",
		"in_voltage0_scale":  "1.0",
		"out_voltage0_raw":   "0",
		"out_voltage0_scale": "1.0",
	})

	dev := g.addDevice(client.IIO{
		ID:          "ID-iio-rule-dev",
		Description: "tank controller",
		Device:      "loop-controller",
		PollPeriod:  iioTestIdle,
	})

	sensor := g.addChannel(client.IIOChannel{
		ID:          "ID-iio-rule-sensor",
		Parent:      "ID-iio-rule-dev",
		Description: "tank level",
		Channel:     "in_voltage0",
		ChannelType: "voltage",
		Direction:   data.PointValueInput,
		Units:       "V",
		Scale:       1,
	})

	valve := g.addChannel(client.IIOChannel{
		ID:          "ID-iio-rule-valve",
		Parent:      "ID-iio-rule-dev",
		Description: "fill valve",
		Channel:     "out_voltage0",
		ChannelType: "voltage",
		Direction:   data.PointValueOutput,
		Units:       "V",
		Scale:       1,
	})

	g.start("ID-iio-rule-dev")

	iioWait(t, dev, iioTestTimeout, "the device to connect",
		func(c client.IIO) bool { return c.Connected })
	iioWait(t, sensor, iioTestTimeout, "the sensor to be read",
		func(c client.IIOChannel) bool { return iioNear(c.Value, 1.0) })

	rule := client.Rule{
		ID:          "ID-iio-rule",
		Parent:      g.root.ID,
		Description: "close the valve when the tank is full",
	}

	if err := client.SendNodeType(g.nc, rule, "test"); err != nil {
		t.Fatal("Error sending rule node: ", err)
	}

	cond := client.Condition{
		ID:            "ID-iio-rule-cond",
		Parent:        rule.ID,
		Description:   "tank above 2 V",
		ConditionType: data.PointValuePointValue,
		PointType:     data.PointTypeValue,
		ValueType:     data.PointValueNumber,
		NodeID:        "ID-iio-rule-sensor",
		Operator:      data.PointValueGreaterThan,
		Value:         2,
	}

	if err := client.SendNodeType(g.nc, cond, "test"); err != nil {
		t.Fatal("Error sending condition node: ", err)
	}

	action := client.Action{
		ID:          "ID-iio-rule-action",
		Parent:      rule.ID,
		Description: "close the valve",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValueSet,
		NodeID:      "ID-iio-rule-valve",
		Value:       3,
	}

	if err := client.SendNodeType(g.nc, action, "test"); err != nil {
		t.Fatal("Error sending action node: ", err)
	}

	// see the note in rule_test.go: the manager needs a moment before the
	// inactive action is sent, or it does not see the points
	time.Sleep(100 * time.Millisecond)

	actionInactive := client.ActionInactive{
		ID:          "ID-iio-rule-action-inactive",
		Parent:      rule.ID,
		Description: "open the valve",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValueSet,
		NodeID:      "ID-iio-rule-valve",
		Value:       0,
	}

	if err := client.SendNodeType(g.nc, actionInactive, "test"); err != nil {
		t.Fatal("Error sending inactive action node: ", err)
	}

	// wait for the rule client to pick up its children
	time.Sleep(250 * time.Millisecond)

	g.write(dir, "in_voltage0_raw", "3000")

	iioWait(t, sensor, iioTestTimeout, "the tank to read full",
		func(c client.IIOChannel) bool { return iioNear(c.Value, 3.0) })

	iioWait(t, valve, iioTestTimeout, "the rule to close the valve",
		func(c client.IIOChannel) bool { return iioNear(c.Value, 3.0) })

	if got := g.read(dir, "out_voltage0_raw"); got != "3000" {
		t.Errorf("Expected the valve channel to be driven to 3000, got %v", got)
	}

	g.write(dir, "in_voltage0_raw", "1000")

	iioWait(t, sensor, iioTestTimeout, "the tank to drain",
		func(c client.IIOChannel) bool { return iioNear(c.Value, 1.0) })

	iioWait(t, valve, iioTestTimeout, "the rule to open the valve",
		func(c client.IIOChannel) bool { return iioNear(c.Value, 0) })
}
