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
// were built for. The node must stay owned by the device's boundary: a device
// replicates only its own boundary's streams, so if the mirror moved ownership
// to the upstream root, a valueSet written on the mirror would be stored where
// the device never reads it and the hardware would never see the command.
func TestSyncMirrorAcrossBoundary(t *testing.T) {
	ncU, rootU, stopU, err := server.TestServer("2")
	if err != nil {
		t.Fatal("Error starting upstream test server: ", err)
	}
	defer stopU()

	ncD, rootD, stopD, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting downstream test server: ", err)
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

	gpioID := "gpio-cross"
	err = client.SendNode(ncD, data.NodeEdge{
		ID: gpioID, Type: data.NodeTypeGPIO, Parent: rootD.ID,
		Points: data.Points{
			data.NewPointString(data.PointTypeDescription, "", "pump enable"),
			data.NewPointString(data.PointTypeChip, "", "sim"),
			data.NewPointString(data.PointTypeLine, "", "5"),
		},
	}, "test")
	if err != nil {
		t.Fatal(err)
	}

	waitFor(t, 10*time.Second, "gpio node not synced upstream", func() bool {
		nodes, err := client.GetNodes(ncU, rootD.ID, gpioID, "", false)
		return err == nil && len(nodes) > 0
	})

	groupU := "group-up"
	err = client.SendNode(ncU, data.NodeEdge{ID: groupU, Type: data.NodeTypeGroup, Parent: rootU.ID,
		Points: data.Points{data.NewPointString(data.PointTypeDescription, "", "customer group")}}, "test")
	if err != nil {
		t.Fatal(err)
	}

	// the mirror is what previously flipped the node's owning boundary
	fmt.Println("**** mirror the device's gpio node into an upstream group")
	if err := client.MirrorNode(ncU, gpioID, rootD.ID, groupU, "test"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)

	descOn := func(nc *nats.Conn) string {
		nodes, err := client.GetNodes(nc, "all", gpioID, "", false)
		if err != nil || len(nodes) == 0 {
			return "<none>"
		}
		d, _ := nodes[0].Points.Text(data.PointTypeDescription, "")
		return d
	}

	arrives := func(nc *nats.Conn, want string) bool {
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			if descOn(nc) == want {
				return true
			}
			time.Sleep(300 * time.Millisecond)
		}
		return false
	}

	fmt.Println("**** device -> upstream")
	if err := client.SendNodePoint(ncD, gpioID,
		data.NewPointString(data.PointTypeDescription, "", "from-device"), true); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("     reached upstream: %v\n", arrives(ncU, "from-device"))

	fmt.Println("**** upstream -> device (this is the valueSet path)")
	if err := client.SendNodePoint(ncU, gpioID,
		data.NewPointString(data.PointTypeDescription, "", "from-upstream"), true); err != nil {
		t.Fatal(err)
	}
	ok := arrives(ncD, "from-upstream")
	fmt.Printf("     reached device:   %v\n", ok)

	fmt.Println("**** valueSet written on the MIRROR, does the device see it?")
	if err := client.SendNodePoint(ncU, gpioID,
		data.NewPointFloat(data.PointTypeValueSet, "", 1), true); err != nil {
		t.Fatal(err)
	}
	vsDeadline := time.Now().Add(30 * time.Second)
	vsOK := false
	for time.Now().Before(vsDeadline) {
		nodes, err := client.GetNodes(ncD, "all", gpioID, "", false)
		if err == nil && len(nodes) > 0 {
			if v, found := nodes[0].Points.Value(data.PointTypeValueSet, ""); found && v == 1 {
				vsOK = true
				break
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	fmt.Printf("     valueSet reached device: %v\n", vsOK)

	if !ok || !vsOK {
		t.Errorf("upstream writes did not reach the device (point=%v valueSet=%v)", ok, vsOK)
	}
}
