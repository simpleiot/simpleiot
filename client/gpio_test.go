package client_test

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// gpioTestTimeout is how long a case waits for a point to reach the node.
// Everything but the request retry happens in milliseconds; the margin is
// there so a loaded build machine does not fail the test.
const gpioTestTimeout = time.Second * 2

// gpioTest is a SIOT test server with the helpers the GPIO cases share. Every
// case runs on the simulated chip, so no hardware and no root access is
// involved.
type gpioTest struct {
	t    *testing.T
	nc   *nats.Conn
	root data.NodeEdge
	// stops holds the node watchers to shut down when the test ends
	stops []func()
}

func newGPIOTest(t *testing.T) *gpioTest {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	g := &gpioTest{t: t, nc: nc, root: root}

	t.Cleanup(func() {
		for _, s := range g.stops {
			s()
		}
		stop()
	})

	return g
}

// add creates a gpio node and returns a getter for its current state
func (g *gpioTest) add(cfg client.GPIO) func() client.GPIO {
	g.t.Helper()

	cfg.Parent = g.root.ID

	if err := client.SendNodeType(g.nc, cfg, "test"); err != nil {
		g.t.Fatalf("Error sending %v node: %v", cfg.Description, err)
	}

	get, stop, err := client.NodeWatcher[client.GPIO](g.nc, cfg.ID, cfg.Parent)
	if err != nil {
		g.t.Fatalf("Error setting up watcher for %v: %v", cfg.Description, err)
	}

	g.stops = append(g.stops, stop)

	return get
}

// send publishes points on a node as if they came from the UI or an upstream
// instance. The origin is set so the client manager passes them to the client.
func (g *gpioTest) send(id string, pts ...data.Point) {
	g.t.Helper()

	for i := range pts {
		pts[i].Origin = "test"
	}

	if err := client.SendNodePoints(g.nc, id, pts, true); err != nil {
		g.t.Fatalf("Error sending points to %v: %v", id, err)
	}
}

// gpioOnOff builds a boolean point of the given type
func gpioOnOff(typ string, v bool) data.Point {
	return data.NewPointFloat(typ, "", data.BoolToFloat(v))
}

// gpioWait blocks until the node satisfies cond, and fails the test with the
// node's current state when it does not
func gpioWait(t *testing.T, get func() client.GPIO, timeout time.Duration,
	msg string, cond func(client.GPIO) bool) client.GPIO {
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

// gpioConnected is the condition every case starts from
func gpioConnected(c client.GPIO) bool {
	return c.Connected
}

// TestGPIO exercises the GPIO client against the simulated chip. Each case
// uses its own line offset, because the simulated chip is process wide and a
// line holds its level after it is released.
func TestGPIO(t *testing.T) {
	g := newGPIOTest(t)

	t.Run("output write", func(t *testing.T) {
		out := g.add(client.GPIO{
			ID:          "ID-gpio-out",
			Description: "output",
			Chip:        data.PointValueSim,
			Line:        "1",
			Direction:   data.PointValueOutput,
		})

		gpioWait(t, out, gpioTestTimeout, "output to connect", gpioConnected)

		g.send("ID-gpio-out", gpioOnOff(data.PointTypeValueSet, true))

		gpioWait(t, out, gpioTestTimeout, "output value to go true",
			func(c client.GPIO) bool { return c.Value })
	})

	t.Run("input edge", func(t *testing.T) {
		in := g.add(client.GPIO{
			ID:          "ID-gpio-edge-in",
			Description: "edge input",
			Chip:        data.PointValueSim,
			Line:        "2",
		})

		gpioWait(t, in, gpioTestTimeout, "input to connect", gpioConnected)

		out := g.add(client.GPIO{
			ID:          "ID-gpio-edge-out",
			Description: "edge driver",
			Chip:        data.PointValueSim,
			Line:        "2",
			Direction:   data.PointValueOutput,
		})

		gpioWait(t, out, gpioTestTimeout, "driver to connect", gpioConnected)

		g.send("ID-gpio-edge-out", gpioOnOff(data.PointTypeValueSet, true))

		gpioWait(t, in, gpioTestTimeout, "input to see a rising edge",
			func(c client.GPIO) bool { return c.Value })

		g.send("ID-gpio-edge-out", gpioOnOff(data.PointTypeValueSet, false))

		gpioWait(t, in, gpioTestTimeout, "input to see a falling edge",
			func(c client.GPIO) bool { return !c.Value })
	})

	t.Run("initial value", func(t *testing.T) {
		out := g.add(client.GPIO{
			ID:           "ID-gpio-initial",
			Description:  "initial value",
			Chip:         data.PointValueSim,
			Line:         "3",
			Direction:    data.PointValueOutput,
			InitialValue: true,
		})

		gpioWait(t, out, gpioTestTimeout, "line to read true as soon as it is requested",
			func(c client.GPIO) bool { return c.Connected && c.Value })
	})

	t.Run("active low", func(t *testing.T) {
		// an input on the same offset reads the line level directly, which is
		// what an active low output inverts
		observer := g.add(client.GPIO{
			ID:          "ID-gpio-al-in",
			Description: "active low observer",
			Chip:        data.PointValueSim,
			Line:        "4",
		})

		gpioWait(t, observer, gpioTestTimeout, "observer to connect", gpioConnected)

		out := g.add(client.GPIO{
			ID:          "ID-gpio-al-out",
			Description: "active low output",
			Chip:        data.PointValueSim,
			Line:        "4",
			Direction:   data.PointValueOutput,
			ActiveLow:   true,
		})

		// driving the output inactive raises the line
		gpioWait(t, observer, gpioTestTimeout, "line to go high while the output is inactive",
			func(c client.GPIO) bool { return c.Value })
		gpioWait(t, out, gpioTestTimeout, "output to report inactive",
			func(c client.GPIO) bool { return c.Connected && !c.Value })

		g.send("ID-gpio-al-out", gpioOnOff(data.PointTypeValueSet, true))

		gpioWait(t, out, gpioTestTimeout, "output to report active",
			func(c client.GPIO) bool { return c.Value })
		gpioWait(t, observer, gpioTestTimeout, "line to go low while the output is active",
			func(c client.GPIO) bool { return !c.Value })
	})

	t.Run("line by name", func(t *testing.T) {
		in := g.add(client.GPIO{
			ID:          "ID-gpio-name",
			Description: "line by name",
			Chip:        data.PointValueSim,
			Line:        "sim5",
		})

		gpioWait(t, in, gpioTestTimeout, "the resolved line to be published",
			func(c client.GPIO) bool {
				return c.Connected && c.LineOffset == 5 && c.LineName == "sim5"
			})
	})

	t.Run("polled input", func(t *testing.T) {
		in := g.add(client.GPIO{
			ID:          "ID-gpio-poll-in",
			Description: "polled input",
			Chip:        data.PointValueSim,
			Line:        "6",
			PollPeriod:  20,
		})

		gpioWait(t, in, gpioTestTimeout, "polled input to connect", gpioConnected)

		out := g.add(client.GPIO{
			ID:          "ID-gpio-poll-out",
			Description: "poll driver",
			Chip:        data.PointValueSim,
			Line:        "6",
			Direction:   data.PointValueOutput,
		})

		gpioWait(t, out, gpioTestTimeout, "driver to connect", gpioConnected)

		g.send("ID-gpio-poll-out", gpioOnOff(data.PointTypeValueSet, true))

		// the input asked for no edge events, so this can only arrive by poll
		gpioWait(t, in, gpioTestTimeout, "the poll to pick up the change",
			func(c client.GPIO) bool { return c.Value })
	})

	t.Run("request failure", func(t *testing.T) {
		bad := g.add(client.GPIO{
			ID:          "ID-gpio-bad",
			Description: "no such line",
			Chip:        data.PointValueSim,
			Line:        "nosuchline",
		})

		// the retry backoff starts around a second, so a second failure shows
		// the client is still trying
		gpioWait(t, bad, time.Second*6, "the request to fail and be retried",
			func(c client.GPIO) bool {
				return !c.Connected && c.Error != "" && c.ErrorCount >= 2
			})

		// naming a line that exists recovers without touching the node again
		g.send("ID-gpio-bad", data.NewPointString(data.PointTypeLine, "", "7"))

		gpioWait(t, bad, gpioTestTimeout, "the line to be requested once it is valid",
			func(c client.GPIO) bool { return c.Connected && c.Error == "" })
	})

	t.Run("reconfigure", func(t *testing.T) {
		io := g.add(client.GPIO{
			ID:          "ID-gpio-reconfig",
			Description: "reconfigured line",
			Chip:        data.PointValueSim,
			Line:        "8",
		})

		gpioWait(t, io, gpioTestTimeout, "input to connect", gpioConnected)

		g.send("ID-gpio-reconfig",
			data.NewPointString(data.PointTypeDirection, "", data.PointValueOutput))

		gpioWait(t, io, gpioTestTimeout, "the direction change to land",
			func(c client.GPIO) bool { return c.Direction == data.PointValueOutput })

		g.send("ID-gpio-reconfig", gpioOnOff(data.PointTypeValueSet, true))

		gpioWait(t, io, gpioTestTimeout, "the reconfigured line to be driven",
			func(c client.GPIO) bool { return c.Value && c.Error == "" })
	})

	t.Run("write to an input", func(t *testing.T) {
		in := g.add(client.GPIO{
			ID:          "ID-gpio-write-input",
			Description: "input written to",
			Chip:        data.PointValueSim,
			Line:        "9",
		})

		gpioWait(t, in, gpioTestTimeout, "input to connect", gpioConnected)

		g.send("ID-gpio-write-input", gpioOnOff(data.PointTypeValueSet, true))

		cur := gpioWait(t, in, gpioTestTimeout, "the write to be reported as an error",
			func(c client.GPIO) bool { return c.Error != "" && c.ErrorCount >= 1 })

		if cur.Value {
			t.Errorf("Input line changed on a write, node is: %+v", cur)
		}
	})

	t.Run("disable", func(t *testing.T) {
		out := g.add(client.GPIO{
			ID:          "ID-gpio-disable",
			Description: "disabled line",
			Chip:        data.PointValueSim,
			Line:        "10",
			Direction:   data.PointValueOutput,
		})

		gpioWait(t, out, gpioTestTimeout, "output to connect", gpioConnected)

		g.send("ID-gpio-disable", gpioOnOff(data.PointTypeDisabled, true))

		gpioWait(t, out, gpioTestTimeout, "the line to be released",
			func(c client.GPIO) bool { return !c.Connected })
	})
}

// TestGPIORule wires the loop someone would actually build: an input line
// drives a rule condition, and the rule action drives an output line.
func TestGPIORule(t *testing.T) {
	g := newGPIOTest(t)

	// the driver stands in for whatever is wired to the sensor line
	driver := g.add(client.GPIO{
		ID:          "ID-gpio-rule-driver",
		Description: "rule driver",
		Chip:        data.PointValueSim,
		Line:        "11",
		Direction:   data.PointValueOutput,
	})

	sensor := g.add(client.GPIO{
		ID:          "ID-gpio-rule-sensor",
		Description: "float switch",
		Chip:        data.PointValueSim,
		Line:        "11",
	})

	actuator := g.add(client.GPIO{
		ID:          "ID-gpio-rule-actuator",
		Description: "pump enable",
		Chip:        data.PointValueSim,
		Line:        "12",
		Direction:   data.PointValueOutput,
	})

	gpioWait(t, driver, gpioTestTimeout, "driver to connect", gpioConnected)
	gpioWait(t, sensor, gpioTestTimeout, "sensor to connect", gpioConnected)
	gpioWait(t, actuator, gpioTestTimeout, "actuator to connect", gpioConnected)

	rule := client.Rule{
		ID:          "ID-gpio-rule",
		Parent:      g.root.ID,
		Description: "run the pump when the float switch closes",
	}

	if err := client.SendNodeType(g.nc, rule, "test"); err != nil {
		t.Fatal("Error sending rule node: ", err)
	}

	cond := client.Condition{
		ID:            "ID-gpio-rule-cond",
		Parent:        rule.ID,
		Description:   "float switch closed",
		ConditionType: data.PointValuePointValue,
		PointType:     data.PointTypeValue,
		ValueType:     data.PointValueOnOff,
		NodeID:        "ID-gpio-rule-sensor",
		Operator:      data.PointValueEqual,
		Value:         1,
	}

	if err := client.SendNodeType(g.nc, cond, "test"); err != nil {
		t.Fatal("Error sending condition node: ", err)
	}

	action := client.Action{
		ID:          "ID-gpio-rule-action",
		Parent:      rule.ID,
		Description: "enable the pump",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValueSet,
		NodeID:      "ID-gpio-rule-actuator",
		Value:       1,
	}

	if err := client.SendNodeType(g.nc, action, "test"); err != nil {
		t.Fatal("Error sending action node: ", err)
	}

	// see the note in rule_test.go: the manager needs a moment before the
	// inactive action is sent, or it does not see the points
	time.Sleep(100 * time.Millisecond)

	actionInactive := client.ActionInactive{
		ID:          "ID-gpio-rule-action-inactive",
		Parent:      rule.ID,
		Description: "disable the pump",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValueSet,
		NodeID:      "ID-gpio-rule-actuator",
		Value:       0,
	}

	if err := client.SendNodeType(g.nc, actionInactive, "test"); err != nil {
		t.Fatal("Error sending inactive action node: ", err)
	}

	// wait for the rule client to pick up its children
	time.Sleep(250 * time.Millisecond)

	g.send("ID-gpio-rule-driver", gpioOnOff(data.PointTypeValueSet, true))

	gpioWait(t, sensor, gpioTestTimeout, "sensor to see the line close",
		func(c client.GPIO) bool { return c.Value })
	gpioWait(t, actuator, gpioTestTimeout, "the rule to drive the pump line",
		func(c client.GPIO) bool { return c.Value })

	g.send("ID-gpio-rule-driver", gpioOnOff(data.PointTypeValueSet, false))

	gpioWait(t, sensor, gpioTestTimeout, "sensor to see the line open",
		func(c client.GPIO) bool { return !c.Value })
	gpioWait(t, actuator, gpioTestTimeout, "the rule to release the pump line",
		func(c client.GPIO) bool { return !c.Value })
}
