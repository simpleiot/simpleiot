package client_test

import (
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

// These tests write points that used to stop the whole instance -- a panic in
// a client goroutine ends the process -- and check that the instance keeps
// running with the problem recorded on the node instead. A test process that
// dies here is the old behavior, so surviving to the assertions is most of
// the proof.

// faultTestServer starts an instance on its own ports, so a run of the
// shared test server elsewhere on the machine is not disturbed
func faultTestServer(t *testing.T) (*nats.Conn, data.NodeEdge, func()) {
	t.Helper()

	opts := server.Options{
		NatsPort:        8960,
		HTTPPort:        "8961",
		NatsMonitorPort: 8962,
		NatsWSPort:      8963,
		NatsServer:      "nats://localhost:8960",
		ID:              "sec21",
		DataDir:         filepath.Join(os.TempDir(), "siot-test-sec21"),
	}

	nc, root, stop, err := server.TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	return nc, root, stop
}

// nodeText reads a text point from a node as the operator would see it
func nodeText(t *testing.T, nc *nats.Conn, nodeID, typ string) string {
	t.Helper()

	nodes, err := client.GetNodes(nc, "all", nodeID, "", false)
	if err != nil {
		t.Fatalf("Error getting node %v: %v", nodeID, err)
	}

	if len(nodes) == 0 {
		return ""
	}

	txt, _ := nodes[0].Points.Text(typ, "")
	return txt
}

// waitForError waits for the node's error point to contain want
func waitForError(t *testing.T, nc *nats.Conn, nodeID, want string) {
	t.Helper()

	var last string
	waitFor(t, 5*time.Second, "error on "+nodeID+" containing "+want, func() bool {
		last = nodeText(t, nc, nodeID, data.PointTypeError)
		return strings.Contains(last, want)
	})
	t.Logf("%v error: %v", nodeID, last)
}

// checkInstanceRunning fails the test if the instance no longer answers
func checkInstanceRunning(t *testing.T, nc *nats.Conn) {
	t.Helper()

	if _, err := client.GetRootNode(nc); err != nil {
		t.Fatal("Instance stopped answering: ", err)
	}
}

// TestFaultMetricsBadSliceKey: a metrics node with a prefix point keyed
// "abc" does not decode. The manager used to use the nil client that came
// back with the error and panic.
func TestFaultMetricsBadSliceKey(t *testing.T) {
	nc, root, stop := faultTestServer(t)
	defer stop()

	m := client.Metrics{
		ID:          "sec21-metrics",
		Parent:      root.ID,
		Description: "bad prefix key",
		Type:        data.PointValueSystem,
		Period:      60,
	}

	ne, err := data.Encode(m)
	if err != nil {
		t.Fatal("Error encoding metrics node: ", err)
	}

	// the node points go in before the edge, so the manager finds the
	// node with the bad point already on it
	ne.Points = append(ne.Points, data.NewPointString(data.PointTypePrefix, "abc", "x"))

	if err := client.SendNode(nc, ne, "test"); err != nil {
		t.Fatal("Error sending metrics node: ", err)
	}

	waitForError(t, nc, m.ID, "not a valid index")
	checkInstanceRunning(t, nc)
}

// TestFaultUpdatePollPeriodZero: a pollPeriod of 0 on a running update node
// used to reach Ticker.Reset and panic.
func TestFaultUpdatePollPeriodZero(t *testing.T) {
	nc, root, stop := faultTestServer(t)
	defer stop()

	u := client.Update{
		ID:          "sec21-update",
		Parent:      root.ID,
		Description: "zero poll period",
		PollPeriod:  5,
	}

	if err := client.SendNodeType(nc, u, "test"); err != nil {
		t.Fatal("Error sending update node: ", err)
	}

	// the client fills in the prefix with the hostname when it starts, so
	// a non-empty prefix means the client is running
	waitFor(t, 5*time.Second, "update client to start", func() bool {
		return nodeText(t, nc, u.ID, data.PointTypePrefix) != ""
	})

	p := data.NewPointFloat(data.PointTypePollPeriod, "0", 0)
	p.Origin = "test"
	if err := client.SendNodePoint(nc, u.ID, p, true); err != nil {
		t.Fatal("Error sending pollPeriod point: ", err)
	}

	// the point reaches the client through a subscription, so give it a
	// moment to be handled
	time.Sleep(500 * time.Millisecond)

	checkInstanceRunning(t, nc)

	// the period is bounded before it reaches the ticker, so the client
	// carries on rather than crashing and being restarted
	if errS := nodeText(t, nc, u.ID, data.PointTypeError); strings.Contains(errS, "panic") {
		t.Fatalf("Update client crashed: %v", errS)
	}
}

// TestFaultSignalGeneratorHugeSampleRate: a sampleRate large enough to make
// the sample interval zero used to reach Ticker.Reset and panic.
func TestFaultSignalGeneratorHugeSampleRate(t *testing.T) {
	nc, root, stop := faultTestServer(t)
	defer stop()

	sg := client.SignalGenerator{
		ID:          "sec21-siggen",
		Parent:      root.ID,
		Description: "huge sample rate",
		SignalType:  "sine",
		Frequency:   1,
		MinValue:    0,
		MaxValue:    10,
		SampleRate:  1e30,
	}

	if err := client.SendNodeType(nc, sg, "test"); err != nil {
		t.Fatal("Error sending signal generator node: ", err)
	}

	waitForError(t, nc, sg.ID, "SampleRate must be at most")
	checkInstanceRunning(t, nc)

	// a rate the generator accepts clears the error
	p := data.NewPointFloat(data.PointTypeSampleRate, "", 10)
	p.Origin = "test"
	if err := client.SendNodePoint(nc, sg.ID, p, true); err != nil {
		t.Fatal("Error sending sampleRate point: ", err)
	}

	waitFor(t, 5*time.Second, "signal generator error to clear", func() bool {
		return nodeText(t, nc, sg.ID, data.PointTypeError) == ""
	})
}
