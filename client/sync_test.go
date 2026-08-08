package client_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// TestSync exercises replication in both directions between a device
// (downstream) and a hub (upstream): adoption, point changes, node
// creation, and deletion.
func TestSync(t *testing.T) {
	ncU, _, stopU, err := server.TestServer("2")
	if err != nil {
		t.Fatal("Error starting upstream test server: ", err)
	}
	defer stopU()

	ncD, rootD, stopD, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting downstream test server: ", err)
	}
	defer stopD()

	fmt.Println("**** create sync node")
	sync := client.Sync{
		ID:          "sync-id",
		Parent:      rootD.ID,
		Description: "sync to up",
		URI:         server.TestServerOptions2.NatsServer,
	}

	err = client.SendNodeType(ncD, sync, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// adoption: device node appears in the upstream tree
	waitFor(t, 10*time.Second, "device node not synced", func() bool {
		nodes, err := client.GetNodes(ncU, "all", rootD.ID, "", false)
		return err == nil && len(nodes) > 0
	})

	fmt.Println("**** update description down")
	err = client.SendNodePoint(ncD, rootD.ID,
		data.NewPointString(data.PointTypeDescription, "", "set down"), true)
	if err != nil {
		t.Fatal("error sending node point: ", err)
	}

	waitFor(t, 10*time.Second, "description not propagated upstream", func() bool {
		nodes, err := client.GetNodesType[client.Device](ncU, "all", rootD.ID)
		return err == nil && len(nodes) > 0 && nodes[0].Description == "set down"
	})

	fmt.Println("**** update description up")
	err = client.SendNodePoint(ncU, rootD.ID,
		data.NewPointString(data.PointTypeDescription, "", "set up"), true)
	if err != nil {
		t.Fatal("error sending node point: ", err)
	}

	waitFor(t, 10*time.Second, "description not propagated downstream", func() bool {
		nodes, err := client.GetNodesType[client.Device](ncD, "all", rootD.ID)
		return err == nil && len(nodes) > 0 && nodes[0].Description == "set up"
	})

	fmt.Println("**** create node down")
	varD := client.Variable{ID: "varDown", Parent: rootD.ID, Description: "varDown"}
	err = client.SendNodeType(ncD, varD, "test")
	if err != nil {
		t.Fatal("Error sending varD: ", err)
	}

	waitFor(t, 10*time.Second, "varDown not propagated upstream", func() bool {
		nodes, err := client.GetNodesType[client.Variable](ncU, "all", "varDown")
		return err == nil && len(nodes) > 0
	})

	fmt.Println("**** create node up")
	varU := client.Variable{ID: "varUp", Parent: rootD.ID, Description: "varUp"}
	err = client.SendNodeType(ncU, varU, "test")
	if err != nil {
		t.Fatal("Error sending varU: ", err)
	}

	waitFor(t, 10*time.Second, "varUp not propagated downstream", func() bool {
		nodes, err := client.GetNodesType[client.Variable](ncD, "all", "varUp")
		return err == nil && len(nodes) > 0
	})

	fmt.Println("**** delete node down")
	err = client.SendEdgePoint(ncD, varD.ID, rootD.ID,
		data.NewPointFloat(data.PointTypeTombstone, "", 1), true)
	if err != nil {
		t.Fatal("error sending edge point: ", err)
	}

	waitFor(t, 10*time.Second, "varDown delete not propagated upstream", func() bool {
		nodes, err := client.GetNodesType[client.Variable](ncU, rootD.ID, varD.ID)
		return err == nil && len(nodes) == 0
	})

	fmt.Println("**** delete node up")
	err = client.SendEdgePoint(ncU, varU.ID, rootD.ID,
		data.NewPointFloat(data.PointTypeTombstone, "", 1), true)
	if err != nil {
		t.Fatal("error sending edge point: ", err)
	}

	waitFor(t, 10*time.Second, "varUp delete not propagated downstream", func() bool {
		nodes, err := client.GetNodesType[client.Variable](ncD, rootD.ID, varU.ID)
		return err == nil && len(nodes) == 0
	})

	fmt.Println("sync test finished")
}

// TestSyncDetachUpstream verifies the Stage 3 detach semantics: when
// the upstream deletes a device node, the device does not force itself
// back into the tree — only the upstream can restore the edge.
func TestSyncDetachUpstream(t *testing.T) {
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

	sync := client.Sync{
		ID:          "sync-id",
		Parent:      rootD.ID,
		Description: "sync to up",
		URI:         server.TestServerOptions2.NatsServer,
	}

	err = client.SendNodeType(ncD, sync, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	waitFor(t, 10*time.Second, "device node not synced", func() bool {
		nodes, err := client.GetNodes(ncU, "all", rootD.ID, "", false)
		return err == nil && len(nodes) > 0
	})

	fmt.Println("**** delete downstream node on upstream (detach)")
	err = client.SendEdgePoint(ncU, rootD.ID, rootU.ID,
		data.NewPointFloat(data.PointTypeTombstone, "", 1), true)
	if err != nil {
		t.Fatal("Error deleting upstream node: ", err)
	}

	// the device must not re-create itself; give it time to try
	time.Sleep(3 * time.Second)

	nodes, err := client.GetNodes(ncU, "all", rootD.ID, "", false)
	if err != nil && err != data.ErrDocumentNotFound {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Fatal("detached device node re-appeared upstream")
	}

	fmt.Println("**** restore device node on upstream")
	err = client.SendEdgePoint(ncU, rootD.ID, rootU.ID,
		data.NewPointFloat(data.PointTypeTombstone, "", 0), true)
	if err != nil {
		t.Fatal("Error restoring upstream node: ", err)
	}

	waitFor(t, 10*time.Second, "device node not restored", func() bool {
		nodes, err := client.GetNodes(ncU, "all", rootD.ID, "", false)
		return err == nil && len(nodes) > 0
	})
}

// TestSyncOfflineCatchup verifies that changes made on both sides while
// the connection is down are delivered after reconnect, via the durable
// replication consumers.
func TestSyncOfflineCatchup(t *testing.T) {
	ncU, _, stopU, err := server.TestServer("2")
	if err != nil {
		t.Fatal("Error starting upstream test server: ", err)
	}
	defer stopU()

	ncD, rootD, stopD, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting downstream test server: ", err)
	}
	defer stopD()

	sync := client.Sync{
		ID:          "sync-id",
		Parent:      rootD.ID,
		Description: "sync to up",
		URI:         server.TestServerOptions2.NatsServer,
	}

	err = client.SendNodeType(ncD, sync, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	waitFor(t, 10*time.Second, "device node not synced", func() bool {
		nodes, err := client.GetNodes(ncU, "all", rootD.ID, "", false)
		return err == nil && len(nodes) > 0
	})

	// make sure the down direction is established too, so the upstream
	// origin stream for our boundary exists before we go offline
	err = client.SendNodePoint(ncU, rootD.ID,
		data.NewPointString(data.PointTypeDescription, "", "pre-offline"), true)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, 10*time.Second, "initial config not propagated downstream", func() bool {
		nodes, err := client.GetNodesType[client.Device](ncD, "all", rootD.ID)
		return err == nil && len(nodes) > 0 && nodes[0].Description == "pre-offline"
	})

	fmt.Println("**** disable sync (go offline)")
	// the manager filters own-node points with an empty Origin (it
	// assumes the client wrote them); a UI edit carries an Origin
	disable := data.NewPointFloat(data.PointTypeDisabled, "", 1)
	disable.Origin = "test"
	err = client.SendNodePoint(ncD, "sync-id", disable, true)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)

	// changes on both sides while offline
	varOff := client.Variable{ID: "varOffline", Parent: rootD.ID, Description: "made offline"}
	err = client.SendNodeType(ncD, varOff, "test")
	if err != nil {
		t.Fatal(err)
	}

	err = client.SendNodePoint(ncU, rootD.ID,
		data.NewPointString(data.PointTypeDescription, "", "offline config"), true)
	if err != nil {
		t.Fatal(err)
	}

	// verify nothing leaks through while offline
	time.Sleep(time.Second)
	nodes, err := client.GetNodesType[client.Variable](ncU, "all", "varOffline")
	if err == nil && len(nodes) > 0 {
		t.Fatal("offline node appeared upstream while disconnected")
	}

	fmt.Println("**** re-enable sync (catch up)")
	enable := data.NewPointFloat(data.PointTypeDisabled, "", 0)
	enable.Origin = "test"
	err = client.SendNodePoint(ncD, "sync-id", enable, true)
	if err != nil {
		t.Fatal(err)
	}

	waitFor(t, 15*time.Second, "offline node not caught up upstream", func() bool {
		nodes, err := client.GetNodesType[client.Variable](ncU, "all", "varOffline")
		return err == nil && len(nodes) > 0
	})

	waitFor(t, 15*time.Second, "offline config not caught up downstream", func() bool {
		nodes, err := client.GetNodesType[client.Device](ncD, "all", rootD.ID)
		return err == nil && len(nodes) > 0 && nodes[0].Description == "offline config"
	})
}
