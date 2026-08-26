package store

import (
	"testing"

	"github.com/simpleiot/simpleiot/data"
)

// testEdge adds an edge to the cache with the given type and tombstone
// state.
func testEdge(ec *EdgeCache, up, down, typ string, tombstone bool) {
	tsVal := 0.0
	if tombstone {
		tsVal = 1
	}
	// the store normalizes point keys to "0" before persisting, so
	// cache entries carry "0" rather than ""
	ec.Set(EdgeEntry{
		Up:   up,
		Down: down,
		Type: typ,
		Points: data.Points{
			data.NewPointFloat(data.PointTypeTombstone, "0", tsVal),
		},
	})
}

// testEdgeRole adds an edge carrying a primary or mirror role point.
func testEdgeRole(ec *EdgeCache, up, down, typ string, role data.EdgeRole) {
	pts := data.Points{data.NewPointFloat(data.PointTypeTombstone, "0", 0)}

	switch role {
	case data.EdgeRolePrimary:
		pts = append(pts, data.NewPointFloat(data.PointTypePrimary, "0", 1))
	case data.EdgeRoleMirror:
		pts = append(pts, data.NewPointFloat(data.PointTypeMirror, "0", 1))
	}

	ec.Set(EdgeEntry{Up: up, Down: down, Type: typ, Points: pts})
}

// TestOwningBoundaryMirrorEdge covers the case the primary and mirror roles
// were built for: a device's sensor mirrored into a group on the upstream
// instance. The device's boundary keeps the node, because a device replicates
// only its own boundary's streams -- if the mirror moved ownership to the root,
// a valueSet written on the mirror would be stored where the device never reads
// it and the hardware would never see the command.
func TestOwningBoundaryMirrorEdge(t *testing.T) {
	ec := NewEdgeCache()

	// R (upstream root) > X (device) > S (sensor, primary)
	// R > G (group) > S (the same sensor, mirrored for access)
	testEdge(ec, "root", "R", data.NodeTypeDevice, false)
	testEdge(ec, "R", "G", data.NodeTypeGroup, false)
	testEdge(ec, "R", "X", data.NodeTypeDevice, false)
	testEdgeRole(ec, "X", "S", data.NodeTypeGPIO, data.EdgeRolePrimary)
	testEdgeRole(ec, "G", "S", data.NodeTypeGPIO, data.EdgeRoleMirror)

	if got := ec.OwningBoundary("S", "R"); got != "X" {
		t.Errorf("mirrored across a boundary: OwningBoundary(S) = %v, want X", got)
	}

	// a mirrored group does not claim what is under it either
	testEdgeRole(ec, "G", "SUB", data.NodeTypeGroup, data.EdgeRoleMirror)
	testEdge(ec, "X", "SUB", data.NodeTypeGroup, false)
	testEdge(ec, "SUB", "S2", data.NodeTypeGPIO, false)

	if got := ec.OwningBoundary("S2", "R"); got != "X" {
		t.Errorf("under a mirrored group: OwningBoundary(S2) = %v, want X", got)
	}

	// with only mirror edges to follow, no boundary is reachable and the
	// node falls back to the instance root
	testEdgeRole(ec, "G", "ORPHAN", data.NodeTypeGPIO, data.EdgeRoleMirror)

	if got := ec.OwningBoundary("ORPHAN", "R"); got != "R" {
		t.Errorf("mirror-only: OwningBoundary(ORPHAN) = %v, want R", got)
	}
}

func TestIsBoundary(t *testing.T) {
	ec := NewEdgeCache()

	// R (root) > G (group) > X (device) > S (sensor)
	testEdge(ec, "root", "R", data.NodeTypeDevice, false)
	testEdge(ec, "R", "G", data.NodeTypeGroup, false)
	testEdge(ec, "G", "X", data.NodeTypeDevice, false)
	testEdge(ec, "X", "S", data.NodeTypeVariable, false)

	if !ec.IsBoundary("R", "R") {
		t.Error("root node must be a boundary")
	}
	if !ec.IsBoundary("X", "R") {
		t.Error("device node must be a boundary")
	}
	if ec.IsBoundary("G", "R") {
		t.Error("group node must not be a boundary")
	}
	if ec.IsBoundary("S", "R") {
		t.Error("sensor node must not be a boundary")
	}
}

func TestOwningBoundarySimple(t *testing.T) {
	ec := NewEdgeCache()

	// R > G > X (device) > S
	testEdge(ec, "root", "R", data.NodeTypeDevice, false)
	testEdge(ec, "R", "G", data.NodeTypeGroup, false)
	testEdge(ec, "G", "X", data.NodeTypeDevice, false)
	testEdge(ec, "X", "S", data.NodeTypeVariable, false)

	tests := []struct {
		node, want string
	}{
		{"R", "R"}, // root owns itself
		{"G", "R"}, // above all device boundaries
		{"X", "X"}, // a boundary node owns itself
		{"S", "X"}, // inside the device boundary
	}

	for _, tt := range tests {
		if got := ec.OwningBoundary(tt.node, "R"); got != tt.want {
			t.Errorf("OwningBoundary(%v) = %v, want %v", tt.node, got, tt.want)
		}
	}
}

func TestOwningBoundaryNested(t *testing.T) {
	ec := NewEdgeCache()

	// R > X (device) > Y (device) > S — nested boundaries; the walk
	// stops at the nearest one
	testEdge(ec, "root", "R", data.NodeTypeDevice, false)
	testEdge(ec, "R", "X", data.NodeTypeDevice, false)
	testEdge(ec, "X", "Y", data.NodeTypeDevice, false)
	testEdge(ec, "Y", "S", data.NodeTypeVariable, false)

	if got := ec.OwningBoundary("S", "R"); got != "Y" {
		t.Errorf("nested: OwningBoundary(S) = %v, want Y", got)
	}
	if got := ec.OwningBoundary("Y", "R"); got != "Y" {
		t.Errorf("nested: OwningBoundary(Y) = %v, want Y", got)
	}
}

func TestOwningBoundaryMultiParent(t *testing.T) {
	ec := NewEdgeCache()

	// S is mirrored under device X and under group G (root boundary):
	// reachable from two boundaries, so the root boundary owns it
	testEdge(ec, "root", "R", data.NodeTypeDevice, false)
	testEdge(ec, "R", "G", data.NodeTypeGroup, false)
	testEdge(ec, "R", "X", data.NodeTypeDevice, false)
	testEdge(ec, "X", "S", data.NodeTypeVariable, false)
	testEdge(ec, "G", "S", data.NodeTypeVariable, false)

	if got := ec.OwningBoundary("S", "R"); got != "R" {
		t.Errorf("multi-boundary: OwningBoundary(S) = %v, want R", got)
	}

	// M is mirrored under two groups that both resolve to the root
	// boundary: a single boundary in the set, so it stays with root
	testEdge(ec, "R", "G2", data.NodeTypeGroup, false)
	testEdge(ec, "G", "M", data.NodeTypeVariable, false)
	testEdge(ec, "G2", "M", data.NodeTypeVariable, false)

	if got := ec.OwningBoundary("M", "R"); got != "R" {
		t.Errorf("multi-parent same boundary: OwningBoundary(M) = %v, want R", got)
	}

	// D is mirrored under device X twice via a child group of X: both
	// paths end at X, so X owns it
	testEdge(ec, "X", "GX", data.NodeTypeGroup, false)
	testEdge(ec, "X", "D", data.NodeTypeVariable, false)
	testEdge(ec, "GX", "D", data.NodeTypeVariable, false)

	if got := ec.OwningBoundary("D", "R"); got != "X" {
		t.Errorf("multi-parent one boundary: OwningBoundary(D) = %v, want X", got)
	}
}

func TestOwningBoundaryTombstoned(t *testing.T) {
	ec := NewEdgeCache()

	// S's only path to device X is tombstoned: no boundary reachable,
	// falls back to the root boundary
	testEdge(ec, "root", "R", data.NodeTypeDevice, false)
	testEdge(ec, "R", "X", data.NodeTypeDevice, false)
	testEdge(ec, "X", "S", data.NodeTypeVariable, true)

	if got := ec.OwningBoundary("S", "R"); got != "R" {
		t.Errorf("tombstoned path: OwningBoundary(S) = %v, want R", got)
	}

	// restore the edge and ownership returns to X
	testEdge(ec, "X", "S", data.NodeTypeVariable, false)
	if got := ec.OwningBoundary("S", "R"); got != "X" {
		t.Errorf("restored path: OwningBoundary(S) = %v, want X", got)
	}
}

// TestBoundaryContractBothSides models both sides of a synced pair: a
// hub whose tree contains a device-type node with ID X, and a
// standalone instance whose root is X. Stage 3 relies on both resolving
// the same ownership for the shared subtree so their stream sets line
// up (see the Stage 3 plan).
func TestBoundaryContractBothSides(t *testing.T) {
	// hub: R > G > X (device) > M (modbus) > S (sensor)
	hub := NewEdgeCache()
	testEdge(hub, "root", "R", data.NodeTypeDevice, false)
	testEdge(hub, "R", "G", data.NodeTypeGroup, false)
	testEdge(hub, "G", "X", data.NodeTypeDevice, false)
	testEdge(hub, "X", "M", data.NodeTypeModbus, false)
	testEdge(hub, "M", "S", data.NodeTypeModbusIO, false)

	// device: X (root) > M (modbus) > S (sensor)
	dev := NewEdgeCache()
	testEdge(dev, "root", "X", data.NodeTypeDevice, false)
	testEdge(dev, "X", "M", data.NodeTypeModbus, false)
	testEdge(dev, "M", "S", data.NodeTypeModbusIO, false)

	// every node in the shared subtree must resolve to boundary X on
	// both instances
	for _, id := range []string{"X", "M", "S"} {
		hubOwner := hub.OwningBoundary(id, "R")
		devOwner := dev.OwningBoundary(id, "X")
		if hubOwner != devOwner {
			t.Errorf("node %v: hub owner %v != device owner %v",
				id, hubOwner, devOwner)
		}
		if hubOwner != "X" {
			t.Errorf("node %v: owner = %v, want X", id, hubOwner)
		}
	}

	// edges are owned by the parent's boundary: the edge attaching X
	// into the hub tree (G -> X) belongs to the hub root boundary, so
	// the device side never needs it
	if got := hub.OwningBoundary("G", "R"); got != "R" {
		t.Errorf("edge G->X owner = %v, want R", got)
	}

	// in-subtree edges (M -> S) resolve to X on both sides
	if hub.OwningBoundary("M", "R") != "X" ||
		dev.OwningBoundary("M", "X") != "X" {
		t.Error("edge M->S must be owned by X on both sides")
	}
}
