package store

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

func newTestJsDb(t *testing.T) (*DbJetStream, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "siot-js-test-*")
	if err != nil {
		t.Fatal("Error creating temp dir:", err)
	}

	opts := &server.Options{
		Port:      -1,
		JetStream: true,
		StoreDir:  tmpDir,
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

	db, err := NewJetStreamDb(nc, "")
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
