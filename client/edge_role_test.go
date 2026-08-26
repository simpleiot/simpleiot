package client_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// addNode creates a node of the given type under parent and returns its ID.
func addNode(t *testing.T, nc *nats.Conn, typ, parent string) string {
	t.Helper()

	id := uuid.New().String()

	err := client.SendNode(nc, data.NodeEdge{
		ID:     id,
		Type:   typ,
		Parent: parent,
		Points: data.Points{
			data.NewPointString(data.PointTypeDescription, "", "test "+typ),
		},
	}, "test")

	if err != nil {
		t.Fatalf("error creating %v node: %v", typ, err)
	}

	return id
}

// edgeRole returns the role of the edge between parent and id.
func edgeRole(t *testing.T, nc *nats.Conn, id, parent string) data.EdgeRole {
	t.Helper()

	nodes, err := client.GetNodes(nc, parent, id, "", false)
	if err != nil {
		t.Fatalf("error fetching node %v under %v: %v", id, parent, err)
	}

	if len(nodes) != 1 {
		t.Fatalf("expected 1 edge for %v under %v, got %v", id, parent, len(nodes))
	}

	return nodes[0].EdgeRole()
}

// edgeCount returns how many live edges a node has.
func edgeCount(t *testing.T, nc *nats.Conn, id string) int {
	t.Helper()

	nodes, err := client.GetNodes(nc, "all", id, "", false)
	if err != nil {
		t.Fatalf("error fetching edges for %v: %v", id, err)
	}

	return len(nodes)
}

// TestCreateMarksPrimary checks that a node owning something outside the tree
// is created with a primary edge, and that a node with no primary location is
// left unmarked.
func TestCreateMarksPrimary(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	gpioID := addNode(t, nc, data.NodeTypeGPIO, root.ID)

	if role := edgeRole(t, nc, gpioID, root.ID); role != data.EdgeRolePrimary {
		t.Errorf("gpio node created with role %v, expected primary", role)
	}

	groupID := addNode(t, nc, data.NodeTypeGroup, root.ID)

	if role := edgeRole(t, nc, groupID, root.ID); role != data.EdgeRoleNone {
		t.Errorf("group node created with role %v, expected none", role)
	}
}

// TestSendNodeLeavesExistingEdgeAlone checks that updating a node through
// SendNode does not mark an edge that already exists. An import updates
// through the same call, and marking a mirror primary would start a second
// client on it.
func TestSendNodeLeavesExistingEdgeAlone(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	groupID := addNode(t, nc, data.NodeTypeGroup, root.ID)
	gpioID := addNode(t, nc, data.NodeTypeGPIO, root.ID)

	if err := client.MirrorNode(nc, gpioID, root.ID, groupID, "test"); err != nil {
		t.Fatal("error mirroring node: ", err)
	}

	// an update the way an import sends one: same ID and parent, points
	// only, no edge points
	err = client.SendNode(nc, data.NodeEdge{
		ID:     gpioID,
		Type:   data.NodeTypeGPIO,
		Parent: groupID,
		Points: data.Points{
			data.NewPointString(data.PointTypeDescription, "", "updated"),
		},
	}, "test")

	if err != nil {
		t.Fatal("error updating node: ", err)
	}

	if role := edgeRole(t, nc, gpioID, groupID); role != data.EdgeRoleMirror {
		t.Errorf("mirror edge became %v after an update, expected it to stay a mirror", role)
	}
}

// TestMirrorMarksMirror covers mirroring a node that has a primary location,
// mirroring a mirror, and mirroring a node that has none.
func TestMirrorMarksMirror(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	groupA := addNode(t, nc, data.NodeTypeGroup, root.ID)
	groupB := addNode(t, nc, data.NodeTypeGroup, root.ID)

	gpioID := addNode(t, nc, data.NodeTypeGPIO, root.ID)

	if err := client.MirrorNode(nc, gpioID, root.ID, groupA, "test"); err != nil {
		t.Fatal("error mirroring gpio node: ", err)
	}

	if role := edgeRole(t, nc, gpioID, groupA); role != data.EdgeRoleMirror {
		t.Errorf("mirror of gpio node got role %v, expected mirror", role)
	}

	if role := edgeRole(t, nc, gpioID, root.ID); role != data.EdgeRolePrimary {
		t.Errorf("primary edge became %v after mirroring, expected it to stay primary", role)
	}

	// mirroring a mirror produces another mirror, not a second primary
	if err := client.MirrorNode(nc, gpioID, groupA, groupB, "test"); err != nil {
		t.Fatal("error mirroring a mirror: ", err)
	}

	if role := edgeRole(t, nc, gpioID, groupB); role != data.EdgeRoleMirror {
		t.Errorf("mirror of a mirror got role %v, expected mirror", role)
	}

	// a user has no primary location, so mirroring one marks nothing
	userID := addNode(t, nc, data.NodeTypeUser, root.ID)

	if err := client.MirrorNode(nc, userID, root.ID, groupA, "test"); err != nil {
		t.Fatal("error mirroring user: ", err)
	}

	if role := edgeRole(t, nc, userID, groupA); role != data.EdgeRoleNone {
		t.Errorf("mirror of user got role %v, expected none", role)
	}
}

// TestMoveOwnedNode checks that a node found through its parent cannot be
// moved out from under it, while a node that may live anywhere still can.
func TestMoveOwnedNode(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	groupID := addNode(t, nc, data.NodeTypeGroup, root.ID)
	modbusID := addNode(t, nc, data.NodeTypeModbus, root.ID)
	ioID := addNode(t, nc, data.NodeTypeModbusIO, modbusID)

	err = client.MoveNode(nc, ioID, modbusID, groupID, "test")
	if err == nil {
		t.Error("expected moving a modbusIo node into a group to fail")
	}

	if role := edgeRole(t, nc, ioID, modbusID); role != data.EdgeRolePrimary {
		t.Errorf("modbusIo edge is %v after a rejected move, expected primary", role)
	}

	if n := edgeCount(t, nc, ioID); n != 1 {
		t.Errorf("modbusIo has %v edges after a rejected move, expected 1", n)
	}

	// a gpio node has no owning parent type, so it stays freely movable
	gpioID := addNode(t, nc, data.NodeTypeGPIO, root.ID)

	if err := client.MoveNode(nc, gpioID, root.ID, groupID, "test"); err != nil {
		t.Fatal("error moving gpio node into a group: ", err)
	}

	if role := edgeRole(t, nc, gpioID, groupID); role != data.EdgeRolePrimary {
		t.Errorf("gpio edge is %v after a move, expected the role to travel with it", role)
	}

	// a mirror stays a mirror when it is moved somewhere else. Coming out
	// unmarked would start a client on it.
	groupC := addNode(t, nc, data.NodeTypeGroup, root.ID)
	groupD := addNode(t, nc, data.NodeTypeGroup, root.ID)

	if err := client.MirrorNode(nc, gpioID, groupID, groupC, "test"); err != nil {
		t.Fatal("error mirroring node: ", err)
	}

	if err := client.MoveNode(nc, gpioID, groupC, groupD, "test"); err != nil {
		t.Fatal("error moving mirror: ", err)
	}

	if role := edgeRole(t, nc, gpioID, groupD); role != data.EdgeRoleMirror {
		t.Errorf("mirror is %v after a move, expected it to stay a mirror", role)
	}
}

// TestDuplicateMirrorIsPrimary checks that duplicating a mirror produces a new
// node with a primary edge rather than a node whose only edge is a mirror.
func TestDuplicateMirrorIsPrimary(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	groupA := addNode(t, nc, data.NodeTypeGroup, root.ID)
	groupB := addNode(t, nc, data.NodeTypeGroup, root.ID)

	gpioID := addNode(t, nc, data.NodeTypeGPIO, root.ID)

	if err := client.MirrorNode(nc, gpioID, root.ID, groupA, "test"); err != nil {
		t.Fatal("error mirroring node: ", err)
	}

	if err := client.DuplicateNode(nc, gpioID, groupB, "test"); err != nil {
		t.Fatal("error duplicating node: ", err)
	}

	children, err := client.GetNodes(nc, groupB, "all", data.NodeTypeGPIO, false)
	if err != nil {
		t.Fatal("error fetching duplicate: ", err)
	}

	if len(children) != 1 {
		t.Fatalf("expected 1 duplicate under groupB, got %v", len(children))
	}

	if role := children[0].EdgeRole(); role != data.EdgeRolePrimary {
		t.Errorf("duplicate of a mirror got role %v, expected primary", role)
	}
}

// TestDeletePrimaryRemovesMirrors checks that deleting the node where it lives
// takes its mirrors with it, and that deleting a mirror leaves the rest alone.
func TestDeletePrimaryRemovesMirrors(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	groupA := addNode(t, nc, data.NodeTypeGroup, root.ID)
	groupB := addNode(t, nc, data.NodeTypeGroup, root.ID)

	gpioID := addNode(t, nc, data.NodeTypeGPIO, root.ID)

	for _, g := range []string{groupA, groupB} {
		if err := client.MirrorNode(nc, gpioID, root.ID, g, "test"); err != nil {
			t.Fatal("error mirroring node: ", err)
		}
	}

	if n := edgeCount(t, nc, gpioID); n != 3 {
		t.Fatalf("expected 3 edges after mirroring twice, got %v", n)
	}

	// deleting a mirror removes only that mirror
	if err := client.DeleteNode(nc, gpioID, groupA, "test"); err != nil {
		t.Fatal("error deleting mirror: ", err)
	}

	if n := edgeCount(t, nc, gpioID); n != 2 {
		t.Errorf("expected 2 edges after deleting one mirror, got %v", n)
	}

	// deleting the primary takes the remaining mirror with it
	if err := client.DeleteNode(nc, gpioID, root.ID, "test"); err != nil {
		t.Fatal("error deleting primary: ", err)
	}

	if n := edgeCount(t, nc, gpioID); n != 0 {
		t.Errorf("expected 0 edges after deleting the primary, got %v", n)
	}
}

// TestDeleteUnmarkedNodeUnchanged checks that a node with no primary location
// keeps the behavior it has always had: deleting it from one group leaves it
// in the others.
func TestDeleteUnmarkedNodeUnchanged(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	groupA := addNode(t, nc, data.NodeTypeGroup, root.ID)
	userID := addNode(t, nc, data.NodeTypeUser, root.ID)

	if err := client.MirrorNode(nc, userID, root.ID, groupA, "test"); err != nil {
		t.Fatal("error mirroring user: ", err)
	}

	if err := client.DeleteNode(nc, userID, root.ID, "test"); err != nil {
		t.Fatal("error deleting user: ", err)
	}

	if n := edgeCount(t, nc, userID); n != 1 {
		t.Errorf("expected the user to remain in 1 group, got %v edges", n)
	}
}
