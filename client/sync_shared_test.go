package client_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// TestSyncSharedNode covers a node mirrored into several devices: the
// hub's writes reach every device it is mirrored into, a device receives
// the node's current values when the mirror is made, removing a mirror
// removes the node from that device only, and a device's own write
// reaches the hub and not the other device.
func TestSyncSharedNode(t *testing.T) {
	ncH, rootH, stopH, err := server.TestServerOpts(server.TestServerOptions2)
	if err != nil {
		t.Fatal("Error starting hub: ", err)
	}
	defer func() { stopH() }()

	ncA, rootA, stopA, err := server.TestServerOpts(server.TestServerOptions)
	if err != nil {
		t.Fatal("Error starting device A: ", err)
	}
	defer stopA()

	ncB, rootB, stopB, err := server.TestServerOpts(server.TestServerOptions3)
	if err != nil {
		t.Fatal("Error starting device B: ", err)
	}
	defer stopB()

	for _, d := range []struct {
		nc   *nats.Conn
		root data.NodeEdge
	}{{ncA, rootA}, {ncB, rootB}} {
		sync := client.Sync{
			ID:          "sync-" + d.root.ID,
			Parent:      d.root.ID,
			Description: "sync to hub",
			URI:         server.TestServerOptions2.NatsServer,
		}
		err = client.SendNodeType(d.nc, sync, "test")
		if err != nil {
			t.Fatal("Error sending sync node: ", err)
		}
	}

	waitFor(t, 10*time.Second, "devices not adopted by hub", func() bool {
		a, errA := client.GetNodes(ncH, "all", rootA.ID, "", false)
		b, errB := client.GetNodes(ncH, "all", rootB.ID, "", false)
		return errA == nil && errB == nil && len(a) > 0 && len(b) > 0
	})

	value := func(nc *nats.Conn, id string) (float64, bool) {
		nodes, err := client.GetNodesType[client.Variable](nc, "all", id)
		if err != nil || len(nodes) == 0 {
			return 0, false
		}
		v, ok := nodes[0].Value["0"]
		if !ok {
			v, ok = nodes[0].Value[""]
		}
		return v, ok
	}

	waitValue := func(nc *nats.Conn, id string, want float64, what string) {
		t.Helper()
		waitFor(t, 10*time.Second, what, func() bool {
			v, ok := value(nc, id)
			return ok && v == want
		})
	}

	fmt.Println("**** shared variable on the hub, mirrored into both devices")
	shared := client.Variable{ID: "shared", Parent: rootH.ID, Description: "shared",
		Value: map[string]float64{"0": 1}}
	err = client.SendNodeType(ncH, shared, "test")
	if err != nil {
		t.Fatal("Error sending shared: ", err)
	}

	for _, dev := range []string{rootA.ID, rootB.ID} {
		err = client.MirrorNode(ncH, shared.ID, rootH.ID, dev, "test")
		if err != nil {
			t.Fatal("Error mirroring shared: ", err)
		}
	}

	waitValue(ncA, shared.ID, 1, "shared not seeded on A")
	waitValue(ncB, shared.ID, 1, "shared not seeded on B")

	fmt.Println("**** hub write reaches both")
	err = client.SendNodePoint(ncH, shared.ID, data.NewPointFloat(data.PointTypeValue, "", 2), true)
	if err != nil {
		t.Fatal(err)
	}
	waitValue(ncA, shared.ID, 2, "hub write not on A")
	waitValue(ncB, shared.ID, 2, "hub write not on B")

	fmt.Println("**** device write reaches the hub and not the other device")
	err = client.SendNodePoint(ncA, shared.ID, data.NewPointFloat(data.PointTypeValue, "", 3), true)
	if err != nil {
		t.Fatal(err)
	}
	waitValue(ncH, shared.ID, 3, "A's write not on hub")
	time.Sleep(time.Second)
	if v, _ := value(ncB, shared.ID); v != 2 {
		t.Fatalf("A's write reached B: value = %v", v)
	}

	fmt.Println("**** remove the mirror from B")
	err = client.SendEdgePoint(ncH, shared.ID, rootB.ID,
		data.NewPointFloat(data.PointTypeTombstone, "", 1), true)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, 10*time.Second, "shared still on B", func() bool {
		_, ok := value(ncB, shared.ID)
		return !ok
	})

	err = client.SendNodePoint(ncH, shared.ID, data.NewPointFloat(data.PointTypeValue, "", 4), true)
	if err != nil {
		t.Fatal(err)
	}
	waitValue(ncA, shared.ID, 4, "hub write not on A after unmirror")

	fmt.Println("**** variable made on A, mirrored into B from the hub")
	varA := client.Variable{ID: "varA", Parent: rootA.ID, Description: "on A",
		Value: map[string]float64{"0": 10}}
	err = client.SendNodeType(ncA, varA, "test")
	if err != nil {
		t.Fatal(err)
	}
	waitValue(ncH, varA.ID, 10, "varA not on hub")

	err = client.MirrorNode(ncH, varA.ID, rootA.ID, rootB.ID, "test")
	if err != nil {
		t.Fatal("Error mirroring varA: ", err)
	}
	waitValue(ncB, varA.ID, 10, "varA not seeded on B with A's value")

	err = client.SendNodePoint(ncH, varA.ID, data.NewPointFloat(data.PointTypeValue, "", 11), true)
	if err != nil {
		t.Fatal(err)
	}
	waitValue(ncA, varA.ID, 11, "hub write to varA not on A")
	waitValue(ncB, varA.ID, 11, "hub write to varA not on B")

	err = client.SendNodePoint(ncA, varA.ID, data.NewPointFloat(data.PointTypeValue, "", 12), true)
	if err != nil {
		t.Fatal(err)
	}
	waitValue(ncH, varA.ID, 12, "A's write to varA not on hub")
	time.Sleep(time.Second)
	if v, _ := value(ncB, varA.ID); v != 11 {
		t.Fatalf("A's write to varA reached B: value = %v", v)
	}

	fmt.Println("**** hub restart appends nothing to the device streams")
	count := func(nc *nats.Conn, boundary string) uint64 {
		t.Helper()
		js, err := jetstream.New(nc)
		if err != nil {
			t.Fatal(err)
		}
		s, err := js.Stream(context.Background(), fmt.Sprintf("inst_%v_%v", boundary, rootH.ID))
		if err != nil {
			t.Fatal(err)
		}
		info, err := s.Info(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		return info.State.Msgs
	}

	beforeA, beforeB := count(ncH, rootA.ID), count(ncH, rootB.ID)

	stopH()
	ncH, _, stopH, err = server.TestServerOptsKeepStore(server.TestServerOptions2)
	if err != nil {
		t.Fatal("Error restarting hub: ", err)
	}

	waitValue(ncH, varA.ID, 12, "varA not on hub after restart")
	time.Sleep(2 * time.Second)

	if a, b := count(ncH, rootA.ID), count(ncH, rootB.ID); a != beforeA || b != beforeB {
		t.Fatalf("hub restart appended to device streams: A %v -> %v, B %v -> %v",
			beforeA, a, beforeB, b)
	}
}
