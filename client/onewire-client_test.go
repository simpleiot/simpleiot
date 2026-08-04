package client_test

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// TestOneWireClient runs the 1-wire client against a fixture laid out like the
// w1 sysfs tree, and checks that a detected device gets a node, that its
// readings arrive as points, and that a device that cannot be read is counted
// as an error on both the device and the bus.
func TestOneWireClient(t *testing.T) {
	root := t.TempDir()

	const deviceID = "28-000005e2fdc3"

	busDir := filepath.Join(root, "w1_bus_master0", deviceID)
	if err := os.MkdirAll(busDir, 0755); err != nil {
		t.Fatal("Error creating w1 bus dir: ", err)
	}

	devDir := filepath.Join(root, deviceID)
	if err := os.MkdirAll(devDir, 0755); err != nil {
		t.Fatal("Error creating w1 device dir: ", err)
	}

	tempFile := filepath.Join(devDir, "temperature")
	if err := os.WriteFile(tempFile, []byte("23456\n"), 0644); err != nil {
		t.Fatal("Error writing w1 temperature: ", err)
	}

	// point the client at the fixture before any client is constructed
	client.OneWireDevicePath = root

	nc, rootNode, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	bus := client.OneWire{
		ID:          "onewire-bus",
		Parent:      rootNode.ID,
		Description: "test 1-wire bus",
		Index:       0,
		PollPeriod:  50,
	}

	if err := client.SendNodeType(nc, bus, "test"); err != nil {
		t.Fatal("Error creating 1-wire bus: ", err)
	}

	busGet, busStop, err := client.NodeWatcher[client.OneWire](nc, bus.ID, bus.Parent)
	if err != nil {
		t.Fatal("Error watching 1-wire bus: ", err)
	}

	defer busStop()

	// the client should detect the device and create a node for it
	var ioID string

	waitFor(t, modbusTestTimeout, "1-wire device node to be created", func() bool {
		ios, err := client.GetNodes(nc, bus.ID, "all", data.NodeTypeOneWireIO, false)
		if err != nil || len(ios) < 1 {
			return false
		}
		ioID = ios[0].ID
		return true
	})

	ioGet, ioStop, err := client.NodeWatcher[client.OneWireIO](nc, ioID, bus.ID)
	if err != nil {
		t.Fatal("Error watching 1-wire device: ", err)
	}

	defer ioStop()

	waitFor(t, modbusTestTimeout, "1-wire device ID", func() bool {
		return ioGet().DeviceID == deviceID
	})

	waitFor(t, modbusTestTimeout, "1-wire temperature in C", func() bool {
		return math.Abs(ioGet().Value-23.456) < 1e-9
	})

	// switching units should convert the reading
	sendPoint(t, nc, ioID, data.NewPointString(data.PointTypeUnits, "", "F"))

	waitFor(t, modbusTestTimeout, "1-wire temperature in F", func() bool {
		return math.Abs(ioGet().Value-(23.456*1.8+32)) < 1e-9
	})

	// a device that can no longer be read counts against both the device and
	// the bus
	if err := os.Remove(tempFile); err != nil {
		t.Fatal("Error removing w1 temperature: ", err)
	}

	waitFor(t, modbusTestTimeout, "1-wire device error count to rise", func() bool {
		return ioGet().ErrorCount > 0
	})

	waitFor(t, modbusTestTimeout, "1-wire bus error count to rise", func() bool {
		return busGet().ErrorCount > 0
	})

	// resetting the count should zero it and clear the request
	sendPoint(t, nc, bus.ID, data.NewPointFloat(data.PointTypeErrorCountReset, "", 1))

	waitFor(t, modbusTestTimeout, "1-wire bus error count reset to be cleared", func() bool {
		return !busGet().ErrorCountReset
	})
}
