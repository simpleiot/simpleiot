package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/simpleiot/simpleiot/data"
)

func newTestNatsServer(t *testing.T, storeDir string) (*server.Server, *nats.Conn) {
	t.Helper()

	opts := &server.Options{
		Port:      -1,
		JetStream: true,
		StoreDir:  storeDir,
		NoSigs:    true,
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatal("Error creating NATS server:", err)
	}

	ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		t.Fatal("NATS server failed to start")
	}

	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		t.Fatal("Error connecting to NATS:", err)
	}

	return ns, nc
}

func newTestJsDb(t *testing.T) (*DbJetStream, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "siot-js-test-*")
	if err != nil {
		t.Fatal("Error creating temp dir:", err)
	}

	ns, nc := newTestNatsServer(t, tmpDir)

	db, err := NewJetStreamDb(nc, "", JsConfig{})
	if err != nil {
		t.Fatal("Error creating JetStream db:", err)
	}

	cleanup := func() {
		nc.Close()
		ns.Shutdown()
		_ = os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestDbJetStream(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()

	if rootID == "" {
		t.Fatal("Root ID is blank")
	}

	rns, err := db.getNodes(nil, "all", rootID, "", false)
	if err != nil {
		t.Fatal("Error getting root node:", err)
	}

	if len(rns) < 1 {
		t.Fatal("No root nodes returned")
	}

	rn := rns[0]

	if rn.ID == "" {
		t.Fatal("Root node ID is blank")
	}

	// modify a point and see if it changes
	err = db.nodePoints(rootID, data.Points{data.NewPointString(data.PointTypeDescription, "", "root")})
	if err != nil {
		t.Fatal(err)
	}

	rns, err = db.getNodes(nil, "all", rootID, "", false)
	if err != nil {
		t.Fatal("Error getting root node:", err)
	}

	rn = rns[0]

	if rn.Desc() != "root" {
		t.Fatal("Description should have been root, got:", rn.Desc())
	}

	// send an old point and verify it does not change
	err = db.nodePoints(rootID, data.Points{func() data.Point {
		p := data.NewPointString(data.PointTypeDescription, "", "root with old time")
		p.Time = time.Now().Add(-time.Hour)
		return p
	}()})
	if err != nil {
		t.Fatal(err)
	}

	rns, err = db.getNodes(nil, "all", rootID, "", false)
	if err != nil {
		t.Fatal("Error getting root node:", err)
	}
	rn = rns[0]

	if rn.Desc() != "root" {
		t.Fatal("Description should have stayed root, got:", rn.Desc())
	}

	// verify default admin user got set
	children, err := db.getNodes(nil, rootID, "all", "", false)
	if err != nil {
		t.Fatal("children error:", err)
	}

	if len(children) < 1 {
		t.Fatal("did not return any children")
	}

	if children[0].Parent != rootID {
		t.Fatal("Parent not correct:", children[0].Parent)
	}

	// test getNodes API
	adminID := children[0].ID

	adminNodes, err := db.getNodes(nil, rootID, adminID, "", false)
	if err != nil {
		t.Fatal("Error getting admin nodes", err)
	}

	if len(adminNodes) < 1 {
		t.Fatal("did not return admin nodes")
	}

	if adminNodes[0].Type != data.NodeTypeUser {
		t.Fatal("getNodes did not return right node type for user")
	}

	adminNodes, err = db.getNodes(nil, "all", adminID, "", false)
	if err != nil {
		t.Fatal("Error getting admin nodes", err)
	}

	if len(adminNodes) < 1 {
		t.Fatal("did not return admin nodes")
	}

	rootNodes, err := db.getNodes(nil, "root", "all", "", false)
	if err != nil {
		t.Fatal("Error getting root nodes", err)
	}

	if len(rootNodes) < 1 {
		t.Fatal("did not return root nodes")
	}

	if rootNodes[0].ID != rootID {
		t.Fatal("root node ID is not correct")
	}

	// test edge points
	err = db.edgePoints(adminID, rootID, data.Points{data.NewPointString(data.PointTypeRole, "", data.PointValueRoleAdmin)})
	if err != nil {
		t.Fatal("Error sending edge points:", err)
	}

	adminNodes, err = db.getNodes(nil, rootID, adminID, "", false)
	if err != nil {
		t.Fatal("Error getting admin nodes", err)
	}

	p, ok := adminNodes[0].EdgePoints.Find(data.PointTypeRole, "")
	if !ok {
		t.Fatal("point not found")
	}
	if p.Txt() != data.PointValueRoleAdmin {
		t.Fatal("point does not have right value")
	}

	// try two children
	groupNodeID := uuid.New().String()

	err = db.edgePoints(groupNodeID, rootID, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeGroup),
	})
	if err != nil {
		t.Fatal("Error creating group edge", err)
	}

	children, err = db.getNodes(nil, rootID, "all", "", false)
	if err != nil {
		t.Fatal("children error:", err)
	}

	if len(children) < 2 {
		t.Fatal("did not return 2 children")
	}

	// verify getNodes with "all" works
	start := time.Now()
	adminNodes, err = db.getNodes(nil, "all", adminID, "", false)
	fmt.Println("getNodes time:", time.Since(start))
	if err != nil {
		t.Fatal("Error getting admin nodes with all specified:", err)
	}

	if adminNodes[0].Parent != rootID {
		t.Fatal("Parent ID is not correct")
	}

	if len(adminNodes) < 1 {
		t.Fatal("did not return admin nodes")
	}
}

func TestDbJetStreamUserCheck(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	nodes, err := db.userCheck("admin", "admin")
	if err != nil {
		t.Fatal("userCheck returned error:", err)
	}

	if len(nodes) < 1 {
		t.Fatal("userCheck did not return nodes")
	}
}

func TestDbJetStreamUp(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()

	children, err := db.getNodes(nil, rootID, "all", "", false)
	if err != nil {
		t.Fatal("Error getting children")
	}

	if len(children) < 1 {
		t.Fatal("no children")
	}

	childID := children[0].ID

	ups, err := db.up(childID, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(ups) < 1 {
		t.Fatal("No ups for admin user")
	}

	if ups[0] != rootID {
		t.Fatal("ups, wrong ID:", ups[0])
	}

	// try to get ups of root node
	ups, err = db.up(rootID, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(ups) < 1 {
		t.Fatal("No ups for root node")
	}

	if ups[0] != "root" {
		t.Fatal("ups, wrong ID for root:", ups[0])
	}
}

// TestDbJetStreamRestart re-opens the store against existing streams
// after a full NATS server restart, then verifies config points are
// intact, and that a write and read work against the reloaded caches.
func TestDbJetStreamRestart(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "siot-js-test-*")
	if err != nil {
		t.Fatal("Error creating temp dir:", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ns, nc := newTestNatsServer(t, tmpDir)

	db, err := NewJetStreamDb(nc, "", JsConfig{})
	if err != nil {
		t.Fatal("Error creating JetStream db:", err)
	}

	rootID := db.rootNodeID()

	// create a group node with a description
	groupID := uuid.New().String()
	err = db.edgePoints(groupID, rootID, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeGroup),
	})
	if err != nil {
		t.Fatal("Error creating group edge:", err)
	}

	err = db.nodePoints(groupID, data.Points{
		data.NewPointString(data.PointTypeDescription, "", "pump house"),
	})
	if err != nil {
		t.Fatal("Error writing group points:", err)
	}

	// full server restart on the same store directory
	nc.Close()
	ns.Shutdown()
	ns.WaitForShutdown()

	ns, nc = newTestNatsServer(t, tmpDir)
	defer func() {
		nc.Close()
		ns.Shutdown()
	}()

	db, err = NewJetStreamDb(nc, "", JsConfig{})
	if err != nil {
		t.Fatal("Error re-opening JetStream db:", err)
	}

	if db.rootNodeID() != rootID {
		t.Fatalf("root ID changed across restart: %v != %v",
			db.rootNodeID(), rootID)
	}

	// config points must be intact after the restart
	nodes, err := db.getNodes(nil, rootID, groupID, "", false)
	if err != nil {
		t.Fatal("Error getting group node:", err)
	}
	if len(nodes) < 1 {
		t.Fatal("group node not found after restart")
	}
	if nodes[0].Desc() != "pump house" {
		t.Fatal("group description lost after restart, got:", nodes[0].Desc())
	}

	// this was the cache poisoning bug: the first write after restart
	// seeded the cache and the config points vanished until the next
	// restart — write a point, then verify config is still there
	err = db.nodePoints(groupID, data.Points{
		data.NewPointFloat(data.PointTypeValue, "", 42),
	})
	if err != nil {
		t.Fatal("Error writing point after restart:", err)
	}

	nodes, err = db.getNodes(nil, rootID, groupID, "", false)
	if err != nil {
		t.Fatal("Error getting group node after write:", err)
	}
	if nodes[0].Desc() != "pump house" {
		t.Fatal("config points lost after first write, got:", nodes[0].Desc())
	}
	v, _ := nodes[0].Points.Value(data.PointTypeValue, "")
	if v != 42 {
		t.Fatal("new point not visible after write, got:", v)
	}

	// the fresh instance should have a single boundary-origin stream
	// node-<rootID>-<rootID> holding everything
	_, err = db.js.Stream(context.Background(), streamName(rootID, rootID))
	if err != nil {
		t.Fatal("expected stream node-<rootID>-<rootID> to exist:", err)
	}
}

// streamSubjectCount returns how many subjects matching filter exist in
// the named stream, 0 if the stream does not exist.
func streamSubjectCount(t *testing.T, db *DbJetStream, name, filter string) int {
	t.Helper()
	ctx := context.Background()

	s, err := db.js.Stream(ctx, name)
	if err != nil {
		return 0
	}
	info, err := s.Info(ctx, jetstream.WithSubjectFilter(filter))
	if err != nil {
		t.Fatal("Error getting stream info:", err)
	}
	return len(info.State.Subjects)
}

// mkTestNode creates a node with an edge and optional description.
func mkTestNode(t *testing.T, db *DbJetStream, parent, id, typ, desc string) {
	t.Helper()
	err := db.edgePoints(id, parent, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", typ),
	})
	if err != nil {
		t.Fatalf("Error creating %v edge: %v", id, err)
	}
	if desc != "" {
		err = db.nodePoints(id, data.Points{
			data.NewPointString(data.PointTypeDescription, "", desc),
		})
		if err != nil {
			t.Fatalf("Error writing %v points: %v", id, err)
		}
	}
}

func TestDbJetStreamCrossBoundaryMove(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()

	devX := uuid.New().String()
	devY := uuid.New().String()
	sensor := uuid.New().String()

	mkTestNode(t, db, rootID, devX, data.NodeTypeDevice, "device X")
	mkTestNode(t, db, rootID, devY, data.NodeTypeDevice, "device Y")
	mkTestNode(t, db, devX, sensor, data.NodeTypeVariable, "sensor 1")

	// capture the original description timestamp; it must survive the
	// move unchanged
	nodes, err := db.getNodes(nil, devX, sensor, "", false)
	if err != nil || len(nodes) < 1 {
		t.Fatal("Error getting sensor:", err)
	}
	descBefore, ok := nodes[0].Points.Find(data.PointTypeDescription, "")
	if !ok {
		t.Fatal("description point missing")
	}

	// sensor subjects live in device X's boundary stream
	xStream := streamName(devX, rootID)
	yStream := streamName(devY, rootID)
	if streamSubjectCount(t, db, xStream, "node."+devX+"."+rootID+"."+sensor+".>") == 0 {
		t.Fatal("sensor subjects not in device X stream before move")
	}

	// move: add edge under Y, tombstone edge under X
	err = db.edgePoints(sensor, devY, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeVariable),
	})
	if err != nil {
		t.Fatal("Error adding new edge:", err)
	}
	err = db.edgePoints(sensor, devX, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 1),
	})
	if err != nil {
		t.Fatal("Error tombstoning old edge:", err)
	}

	// sensor subjects now live in device Y's stream only
	if streamSubjectCount(t, db, yStream, "node."+devY+"."+rootID+"."+sensor+".>") == 0 {
		t.Fatal("sensor subjects not in device Y stream after move")
	}
	if n := streamSubjectCount(t, db, xStream, "node."+devX+"."+rootID+"."+sensor+".>"); n != 0 {
		t.Fatal("sensor subjects remain in device X stream after move:", n)
	}

	// node reads back under Y with the original point timestamp
	nodes, err = db.getNodes(nil, devY, sensor, "", false)
	if err != nil || len(nodes) < 1 {
		t.Fatal("Error getting sensor under Y:", err)
	}
	descAfter, ok := nodes[0].Points.Find(data.PointTypeDescription, "")
	if !ok {
		t.Fatal("description point missing after move")
	}
	if !descAfter.Time.Equal(descBefore.Time) {
		t.Fatalf("description timestamp changed in move: %v != %v",
			descAfter.Time, descBefore.Time)
	}

	// the old edge is tombstoned, not gone
	all, err := db.getNodes(nil, "all", sensor, "", true)
	if err != nil {
		t.Fatal("Error getting all sensor edges:", err)
	}
	if len(all) != 2 {
		t.Fatal("expected 2 edges for moved node, got:", len(all))
	}
}

func TestDbJetStreamSubtreeMoveIntoBoundary(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()

	devX := uuid.New().String()
	group := uuid.New().String()
	sensor := uuid.New().String()

	mkTestNode(t, db, rootID, devX, data.NodeTypeDevice, "device X")
	mkTestNode(t, db, rootID, group, data.NodeTypeGroup, "pump group")
	mkTestNode(t, db, group, sensor, data.NodeTypeVariable, "pump sensor")

	rootStream := streamName(rootID, rootID)
	xStream := streamName(devX, rootID)

	// group and sensor start in the root boundary stream
	if streamSubjectCount(t, db, rootStream, "node."+rootID+"."+rootID+"."+sensor+".>") == 0 {
		t.Fatal("sensor subjects not in root stream before move")
	}

	// move the group (and its subtree) under device X
	err := db.edgePoints(group, devX, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeGroup),
	})
	if err != nil {
		t.Fatal("Error adding group edge under device:", err)
	}
	err = db.edgePoints(group, rootID, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 1),
	})
	if err != nil {
		t.Fatal("Error tombstoning group root edge:", err)
	}

	// the whole subtree moved: group points, group->sensor edge, and
	// sensor points are all in device X's stream now
	for _, filter := range []string{
		"node." + devX + "." + rootID + "." + group + ".p.>",
		"node." + devX + "." + rootID + "." + group + ".ep." + sensor,
		"node." + devX + "." + rootID + "." + sensor + ".p.>",
	} {
		if streamSubjectCount(t, db, xStream, filter) == 0 {
			t.Fatal("expected subjects in device stream after move:", filter)
		}
	}

	// and purged from the root stream (the tombstoned root->group edge
	// stays: it belongs to the root boundary)
	for _, filter := range []string{
		"node." + rootID + "." + rootID + "." + group + ".>",
		"node." + rootID + "." + rootID + "." + sensor + ".>",
	} {
		if n := streamSubjectCount(t, db, rootStream, filter); n != 0 {
			t.Fatal("subjects remain in root stream after move:", filter)
		}
	}

	// reads stay consistent through the move
	nodes, err := db.getNodes(nil, group, sensor, "", false)
	if err != nil || len(nodes) < 1 {
		t.Fatal("Error getting sensor after move:", err)
	}
	if nodes[0].Desc() != "pump sensor" {
		t.Fatal("sensor description lost in move:", nodes[0].Desc())
	}
}

// TestDbJetStreamRetention verifies per-subject retention drops old
// messages of a frequently-written subject while preserving the tips
// of rarely-written subjects (config points).
func TestDbJetStreamRetention(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "siot-js-test-*")
	if err != nil {
		t.Fatal("Error creating temp dir:", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ns, nc := newTestNatsServer(t, tmpDir)
	defer func() {
		nc.Close()
		ns.Shutdown()
	}()

	db, err := NewJetStreamDb(nc, "", JsConfig{MaxMsgsPerSubject: 5})
	if err != nil {
		t.Fatal("Error creating JetStream db:", err)
	}

	rootID := db.rootNodeID()

	// description is written once (a rarely-written config subject)
	err = db.nodePoints(rootID, data.Points{
		data.NewPointString(data.PointTypeDescription, "", "retention test"),
	})
	if err != nil {
		t.Fatal(err)
	}

	// value is written many times (a fast-changing subject)
	base := time.Now()
	for i := range 20 {
		p := data.NewPointFloat(data.PointTypeValue, "", float64(i))
		p.Time = base.Add(time.Duration(i) * time.Millisecond)
		err = db.nodePoints(rootID, data.Points{p})
		if err != nil {
			t.Fatal(err)
		}
	}

	ctx := context.Background()
	s, err := db.js.Stream(ctx, streamName(rootID, rootID))
	if err != nil {
		t.Fatal("Error getting stream:", err)
	}

	valueSubject := nodePointSubject(rootID, rootID, rootID, data.PointTypeValue, "0")
	info, err := s.Info(ctx, jetstream.WithSubjectFilter(valueSubject))
	if err != nil {
		t.Fatal("Error getting stream info:", err)
	}
	if n := info.State.Subjects[valueSubject]; n > 5 {
		t.Fatal("retention did not limit value subject, count:", n)
	}

	// tip of the fast subject is the latest write
	msg, err := s.GetLastMsgForSubject(ctx, valueSubject)
	if err != nil {
		t.Fatal("Error getting value tip:", err)
	}
	pts, err := data.DecodePoints(msg.Data)
	if err != nil || len(pts) < 1 {
		t.Fatal("Error decoding value tip:", err)
	}
	if pts[0].Val() != 19 {
		t.Fatal("value tip is not the latest write:", pts[0].Val())
	}

	// the rarely-written config subject is preserved
	descSubject := nodePointSubject(rootID, rootID, rootID, data.PointTypeDescription, "0")
	_, err = s.GetLastMsgForSubject(ctx, descSubject)
	if err != nil {
		t.Fatal("description tip lost under retention:", err)
	}
}

func TestDbJetStreamTombstoneDeleteUndelete(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()
	groupID := uuid.New().String()

	mkTestNode(t, db, rootID, groupID, data.NodeTypeGroup, "doomed group")

	// delete
	err := db.edgePoints(groupID, rootID, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 1),
	})
	if err != nil {
		t.Fatal("Error tombstoning group:", err)
	}

	nodes, err := db.getNodes(nil, rootID, groupID, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Fatal("deleted node still returned without includeDel")
	}

	nodes, err = db.getNodes(nil, rootID, groupID, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatal("deleted node not returned with includeDel")
	}

	// undelete
	err = db.edgePoints(groupID, rootID, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
	})
	if err != nil {
		t.Fatal("Error undeleting group:", err)
	}

	nodes, err = db.getNodes(nil, rootID, groupID, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatal("undeleted node not returned")
	}
	if nodes[0].Desc() != "doomed group" {
		t.Fatal("node points lost across delete/undelete:", nodes[0].Desc())
	}
}

func TestDbJetStreamReset(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()
	groupID := uuid.New().String()
	mkTestNode(t, db, rootID, groupID, data.NodeTypeGroup, "temporary")

	err := db.reset()
	if err != nil {
		t.Fatal("Error resetting store:", err)
	}

	if db.rootNodeID() != rootID {
		t.Fatal("root ID not preserved across reset")
	}

	nodes, err := db.getNodes(nil, rootID, groupID, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Fatal("group survived reset")
	}

	// admin user is re-created
	users, err := db.userCheck("admin", "admin")
	if err != nil || len(users) < 1 {
		t.Fatal("admin user missing after reset:", err)
	}
}

func TestDbJetStreamEdgeOlderTimestamp(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()
	groupID := uuid.New().String()
	mkTestNode(t, db, rootID, groupID, data.NodeTypeGroup, "group")

	now := time.Now()

	p := data.NewPointString(data.PointTypeRole, "", "current")
	p.Time = now
	err := db.edgePoints(groupID, rootID, data.Points{p})
	if err != nil {
		t.Fatal(err)
	}

	// an older incoming edge point must not replace the tip
	old := data.NewPointString(data.PointTypeRole, "", "stale")
	old.Time = now.Add(-time.Hour)
	err = db.edgePoints(groupID, rootID, data.Points{old})
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := db.getNodes(nil, rootID, groupID, "", false)
	if err != nil || len(nodes) < 1 {
		t.Fatal("Error getting group:", err)
	}
	role, _ := nodes[0].EdgePoints.Text(data.PointTypeRole, "")
	if role != "current" {
		t.Fatal("older edge point replaced newer tip:", role)
	}
}

func TestDbJetStreamBatchPoints(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()

	now := time.Now()

	pts := data.Points{
		{Time: now, Type: data.PointTypeValue},
		{Time: now.Add(-time.Second), Type: data.PointTypeValue},
		{Time: now.Add(-time.Second * 2), Type: data.PointTypeValue},
	}

	err := db.nodePoints(rootID, pts)
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := db.getNodes(nil, "all", rootID, "", false)
	if err != nil {
		t.Fatal("Error getting root node:", err)
	}

	n := nodes[0]

	// After collapse, only one point with the latest time should remain
	var valuePoints data.Points
	for _, p := range n.Points {
		if p.Type == data.PointTypeValue {
			valuePoints = append(valuePoints, p)
		}
	}

	if len(valuePoints) != 1 {
		t.Fatal("Error, point did not get merged, got:", len(valuePoints))
	}

	if !valuePoints[0].Time.Equal(now) {
		t.Fatal("Point collapsing did not pick latest")
	}
}
