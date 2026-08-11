package client_test

import (
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// TestPointSubjectTokens covers the store rejecting points it cannot publish.
// A point travels on a subject ending in its type and key, and listeners read
// the node ID and parent ID from fixed positions in that subject, so a key
// carrying a period would deliver the point to the wrong handler.
func TestPointSubjectTokens(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	// a key the sender should fix. This is the shape a Zephyr board sent:
	// a rail named for its voltage.
	bad := data.NewPointFloat(data.PointTypeTemperature, "PCIe_bridge_0.95V", 88)

	err = client.SendNodePoint(nc, root.ID, bad, true)
	if err == nil {
		t.Fatal("Expected a point with a period in the key to be rejected")
	}

	if !strings.Contains(err.Error(), bad.Key) {
		t.Errorf("Expected the error to name the key %v, got: %v", bad.Key, err)
	}

	if pointExists(t, nc, root.ID, bad.Type, bad.Key) {
		t.Error("A rejected point was stored anyway")
	}

	// the sender is told on the node as well, so a device sending a bad key
	// can be found without reading the server log
	if !waitForPoint(t, nc, root.ID, data.PointTypeError, "") {
		t.Error("Expected an error point on the node describing the rejection")
	}

	// a point that fits in a subject is unaffected
	good := data.NewPointFloat(data.PointTypeTemperature, "PCIe_bridge", 88)

	err = client.SendNodePoint(nc, root.ID, good, true)
	if err != nil {
		t.Fatal("Error sending point: ", err)
	}

	if !pointExists(t, nc, root.ID, good.Type, good.Key) {
		t.Error("A valid point was not stored")
	}
}

func pointExists(t *testing.T, nc *nats.Conn, nodeID, typ, key string) bool {
	t.Helper()

	nodes, err := client.GetNodes(nc, "all", nodeID, "", false)
	if err != nil {
		t.Fatal("Error getting node: ", err)
	}

	if len(nodes) < 1 {
		t.Fatal("Node not found: ", nodeID)
	}

	_, ok := nodes[0].Points.Find(typ, key)

	return ok
}

// waitForPoint polls for a point, which is needed for points the store sends
// on its own rather than in reply to the caller
func waitForPoint(t *testing.T, nc *nats.Conn, nodeID, typ, key string) bool {
	t.Helper()

	start := time.Now()

	for time.Since(start) < time.Second*5 {
		if pointExists(t, nc, nodeID, typ, key) {
			return true
		}

		time.Sleep(time.Millisecond * 50)
	}

	return false
}
