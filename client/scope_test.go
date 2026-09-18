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

// scopeTestServer starts an instance on its own ports, so a run of the
// shared test server elsewhere on the machine is not disturbed
func scopeTestServer(t *testing.T) (*nats.Conn, data.NodeEdge, func()) {
	t.Helper()

	opts := server.Options{
		NatsPort:     8964,
		HTTPPort:     "8965",
		NatsHTTPPort: 8966,
		NatsWSPort:   8967,
		NatsServer:   "nats://localhost:8964",
		ID:           "sec23",
		DataDir:      filepath.Join(os.TempDir(), "siot-test-sec23"),
	}

	nc, root, stop, err := server.TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	return nc, root, stop
}

// nodeValue reads a numeric point from a node
func nodeValue(t *testing.T, nc *nats.Conn, nodeID, typ string) float64 {
	t.Helper()

	nodes, err := client.GetNodes(nc, "all", nodeID, "", false)
	if err != nil {
		t.Fatalf("Error getting node %v: %v", nodeID, err)
	}

	if len(nodes) == 0 {
		return 0
	}

	v, _ := nodes[0].Points.Value(typ, "")
	return v
}

func sendScopeNode[T any](t *testing.T, nc *nats.Conn, n T) {
	t.Helper()

	if err := client.SendNodeType(nc, n, "test"); err != nil {
		t.Fatalf("Error sending node %+v: %v", n, err)
	}
}

// TestRuleActionScope: a rule under group G may write to nodes under G and
// is refused a node under sibling group H, with the refusal recorded on the
// action node. Before this check a rule in one group could rewrite any node
// whose ID it knew.
func TestRuleActionScope(t *testing.T) {
	nc, root, stop := scopeTestServer(t)
	defer stop()

	sendScopeNode(t, nc, client.Group{ID: "sec23-G", Parent: root.ID, Description: "G"})
	sendScopeNode(t, nc, client.Group{ID: "sec23-H", Parent: root.ID, Description: "H"})

	sendScopeNode(t, nc, client.Variable{ID: "sec23-vin", Parent: "sec23-G", Description: "vin"})
	sendScopeNode(t, nc, client.Variable{ID: "sec23-vG", Parent: "sec23-G", Description: "vG"})
	sendScopeNode(t, nc, client.Variable{ID: "sec23-vH", Parent: "sec23-H", Description: "vH"})

	sendScopeNode(t, nc, client.Rule{ID: "sec23-rule", Parent: "sec23-G", Description: "rule in G"})

	sendScopeNode(t, nc, client.Condition{
		ID:            "sec23-cond",
		Parent:        "sec23-rule",
		Description:   "vin high",
		ConditionType: data.PointValuePointValue,
		PointType:     data.PointTypeValue,
		ValueType:     data.PointValueOnOff,
		NodeID:        "sec23-vin",
		Operator:      data.PointValueEqual,
		Value:         1,
	})

	sendScopeNode(t, nc, client.Action{
		ID:          "sec23-act-H",
		Parent:      "sec23-rule",
		Description: "write vH in sibling group",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValue,
		NodeID:      "sec23-vH",
		Value:       1,
	})

	sendScopeNode(t, nc, client.Action{
		ID:          "sec23-act-dot",
		Parent:      "sec23-rule",
		Description: "write an ID that is not a subject token",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValue,
		NodeID:      "sec23-vG.value",
		Value:       1,
	})

	sendScopeNode(t, nc, client.Action{
		ID:          "sec23-act-G",
		Parent:      "sec23-rule",
		Description: "write vG in own group",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValue,
		NodeID:      "sec23-vG",
		Value:       1,
	})

	// a node further down under G is reached by walking up from it
	sendScopeNode(t, nc, client.Group{ID: "sec23-sub", Parent: "sec23-G", Description: "sub"})
	sendScopeNode(t, nc, client.Variable{ID: "sec23-vSub", Parent: "sec23-sub", Description: "vSub"})

	sendScopeNode(t, nc, client.Action{
		ID:          "sec23-act-sub",
		Parent:      "sec23-rule",
		Description: "write vSub in a group under own group",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValue,
		NodeID:      "sec23-vSub",
		Value:       1,
	})

	// let the rule client pick up its actions before triggering it
	time.Sleep(200 * time.Millisecond)

	p := data.NewPointFloat(data.PointTypeValue, "", 1)
	p.Origin = "test"
	if err := client.SendNodePoint(nc, "sec23-vin", p, true); err != nil {
		t.Fatal("Error sending vin point: ", err)
	}

	// the actions under the rule's own group work
	waitFor(t, 5*time.Second, "vG to be written", func() bool {
		return nodeValue(t, nc, "sec23-vG", data.PointTypeValue) == 1
	})
	waitFor(t, 5*time.Second, "vSub to be written", func() bool {
		return nodeValue(t, nc, "sec23-vSub", data.PointTypeValue) == 1
	})

	// the action into the sibling group is refused, with the error on
	// the action node
	var errH string
	waitFor(t, 5*time.Second, "refusal recorded on the H action", func() bool {
		errH = nodeText(t, nc, "sec23-act-H", data.PointTypeError)
		return strings.Contains(errH, "is not under")
	})
	t.Logf("H action error: %v", errH)

	// so is an ID that would change the subject
	var errDot string
	waitFor(t, 5*time.Second, "refusal recorded on the dotted action", func() bool {
		errDot = nodeText(t, nc, "sec23-act-dot", data.PointTypeError)
		return strings.Contains(errDot, "which is not allowed")
	})
	t.Logf("dotted action error: %v", errDot)

	if v := nodeValue(t, nc, "sec23-vH", data.PointTypeValue); v != 0 {
		t.Fatalf("vH in the sibling group was written: %v", v)
	}

	if errG := nodeText(t, nc, "sec23-act-G", data.PointTypeError); errG != "" {
		t.Fatalf("G action has an error: %v", errG)
	}
}

// TestSignalGeneratorDestinationScope: a generator under G is refused a
// destination under sibling group H, with the refusal recorded on its node
func TestSignalGeneratorDestinationScope(t *testing.T) {
	nc, root, stop := scopeTestServer(t)
	defer stop()

	sendScopeNode(t, nc, client.Group{ID: "sec23-G", Parent: root.ID, Description: "G"})
	sendScopeNode(t, nc, client.Group{ID: "sec23-H", Parent: root.ID, Description: "H"})
	sendScopeNode(t, nc, client.Variable{ID: "sec23-vH", Parent: "sec23-H", Description: "vH"})

	sendScopeNode(t, nc, client.SignalGenerator{
		ID:          "sec23-siggen",
		Parent:      "sec23-G",
		Description: "generator in G writing to H",
		SignalType:  "square",
		Frequency:   10,
		MinValue:    1,
		MaxValue:    2,
		SampleRate:  50,
		Destination: client.Destination{NodeID: "sec23-vH"},
	})

	var errS string
	waitFor(t, 5*time.Second, "refusal recorded on the generator", func() bool {
		errS = nodeText(t, nc, "sec23-siggen", data.PointTypeError)
		return strings.Contains(errS, "destination refused")
	})
	t.Logf("generator error: %v", errS)

	// nothing was written into the sibling group
	time.Sleep(200 * time.Millisecond)
	if v := nodeValue(t, nc, "sec23-vH", data.PointTypeValue); v != 0 {
		t.Fatalf("vH in the sibling group was written: %v", v)
	}
}
