package client_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// TestSyncMirrorAcrossBoundary covers a device's hardware node mirrored into a
// group on an upstream instance, which is the case the primary and mirror roles
// were built for. A valueSet written on the mirror has to reach the device and
// be acted on there, and the resulting value has to come back.
//
// The node must stay owned by the device's boundary for this to work. A device
// replicates only its own boundary's streams, so if the mirror moved ownership
// to the upstream root, the valueSet would be stored where the device never
// reads it and the line would never move.
//
// The point carries an Origin, as the UI and rules set one. A point with a
// blank Origin is by contract from the client that owns the node, and the
// client manager filters it; see the Message echo section of
// docs/ref/client.md.
func TestSyncMirrorAcrossBoundary(t *testing.T) {
	ncU, rootU, stopU, err := server.TestServer("2")
	if err != nil {
		t.Fatal(err)
	}
	defer stopU()

	ncD, rootD, stopD, err := server.TestServer()
	if err != nil {
		t.Fatal(err)
	}
	defer stopD()

	sync := client.Sync{ID: "sync-id", Parent: rootD.ID, Description: "sync to up",
		URI: server.TestServerOptions2.NatsServer}
	if err := client.SendNodeType(ncD, sync, "test"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 10*time.Second, "device node not synced", func() bool {
		nodes, err := client.GetNodes(ncU, "all", rootD.ID, "", false)
		return err == nil && len(nodes) > 0
	})

	// a simulated output line on the device
	gpioID := "gpio-actuate"
	if err := client.SendNode(ncD, data.NodeEdge{
		ID: gpioID, Type: data.NodeTypeGPIO, Parent: rootD.ID,
		Points: data.Points{
			data.NewPointString(data.PointTypeDescription, "", "pump relay"),
			data.NewPointString(data.PointTypeChip, "", data.PointValueSim),
			data.NewPointString(data.PointTypeLine, "", "9"),
			data.NewPointString(data.PointTypeDirection, "", data.PointValueOutput),
		},
	}, "test"); err != nil {
		t.Fatal(err)
	}

	waitFor(t, 10*time.Second, "gpio not connected on the device", func() bool {
		ns, err := client.GetNodesType[client.GPIO](ncD, "all", gpioID)
		return err == nil && len(ns) > 0 && ns[0].Connected
	})
	waitFor(t, 10*time.Second, "gpio not synced upstream", func() bool {
		nodes, err := client.GetNodes(ncU, rootD.ID, gpioID, "", false)
		return err == nil && len(nodes) > 0
	})

	groupU := "group-up"
	if err := client.SendNode(ncU, data.NodeEdge{ID: groupU, Type: data.NodeTypeGroup,
		Parent: rootU.ID}, "test"); err != nil {
		t.Fatal(err)
	}
	if err := client.MirrorNode(ncU, gpioID, rootD.ID, groupU, "test"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)

	valueOn := func(nc *nats.Conn) (bool, bool) {
		ns, err := client.GetNodesType[client.GPIO](nc, "all", gpioID)
		if err != nil || len(ns) == 0 {
			return false, false
		}
		return ns[0].Value, ns[0].ValueSet
	}

	v, vs := valueOn(ncD)
	fmt.Printf("before: device value=%v valueSet=%v\n", v, vs)

	// write valueSet on the UPSTREAM the way the UI does, with an origin
	fmt.Println("**** valueSet=1 on the upstream, with Origin set (as the UI writes it)")
	p := data.NewPointFloat(data.PointTypeValueSet, "", 1)
	p.Origin = "some-user-id"
	if err := client.SendNodePoint(ncU, gpioID, p, true); err != nil {
		t.Fatal(err)
	}

	actuated := false
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if v, _ := valueOn(ncD); v {
			actuated = true
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	v, vs = valueOn(ncD)
	fmt.Printf("after (device):   value=%v valueSet=%v  -> line actuated: %v\n", v, vs, actuated)

	// and does the resulting value come back up to the mirror?
	reported := false
	deadline = time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if v, _ := valueOn(ncU); v {
			reported = true
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	v, vs = valueOn(ncU)
	fmt.Printf("after (upstream): value=%v valueSet=%v  -> value reported back: %v\n", v, vs, reported)

	if !actuated {
		t.Error("the GPIO client on the device did not drive the line")
	}
	if !reported {
		t.Error("the resulting value did not reach the mirror")
	}

}

// TestSyncMirrorNoRoleAcrossBoundary covers a node with no primary location --
// a variable, a user -- that lives on a device and is mirrored onto the
// upstream. Neither edge carries a role, because every edge of such a node is
// a real place it lives, so ownership cannot follow a primary edge here.
//
// The node still has to stay owned by the device's boundary. Reaching the
// upstream root as well says only that the node is somewhere in the upstream's
// tree, and if that moved ownership, a value written on the upstream would be
// stored where the device never reads it.
func TestSyncMirrorNoRoleAcrossBoundary(t *testing.T) {
	ncU, rootU, stopU, err := server.TestServer("2")
	if err != nil {
		t.Fatal(err)
	}
	defer stopU()

	ncD, rootD, stopD, err := server.TestServer()
	if err != nil {
		t.Fatal(err)
	}
	defer stopD()

	sync := client.Sync{ID: "sync-id", Parent: rootD.ID, Description: "sync to up",
		URI: server.TestServerOptions2.NatsServer}
	if err := client.SendNodeType(ncD, sync, "test"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 10*time.Second, "device node not synced", func() bool {
		nodes, err := client.GetNodes(ncU, "all", rootD.ID, "", false)
		return err == nil && len(nodes) > 0
	})

	varID := "var-mirrored"
	if err := client.SendNode(ncD, data.NodeEdge{
		ID: varID, Type: data.NodeTypeVariable, Parent: rootD.ID,
		Points: data.Points{
			data.NewPointString(data.PointTypeDescription, "", "Var1"),
			data.NewPointFloat(data.PointTypeValue, "", 0),
		},
	}, "test"); err != nil {
		t.Fatal(err)
	}

	waitFor(t, 10*time.Second, "variable not synced upstream", func() bool {
		nodes, err := client.GetNodes(ncU, rootD.ID, varID, "", false)
		return err == nil && len(nodes) > 0
	})

	// mirror it onto the upstream root, as someone organizing the upstream
	// tree would
	if err := client.MirrorNode(ncU, varID, rootD.ID, rootU.ID, "test"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)

	p := data.NewPointFloat(data.PointTypeValue, "", 42)
	p.Origin = "some-user-id"
	if err := client.SendNodePoint(ncU, varID, p, true); err != nil {
		t.Fatal(err)
	}

	valueOn := func(nc *nats.Conn) float64 {
		nodes, err := client.GetNodes(nc, "all", varID, "", false)
		if err != nil || len(nodes) == 0 {
			return 0
		}
		v, _ := nodes[0].Points.Value(data.PointTypeValue, "")
		return v
	}

	waitFor(t, 30*time.Second, "value written on the mirror did not reach the device",
		func() bool { return valueOn(ncD) == 42 })
}
