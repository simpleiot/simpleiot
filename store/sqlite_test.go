package store

import (
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
)

var testFile = "test.sqlite"

func newTestDb(t *testing.T) *DbSqlite {
	exec.Command("sh", "-c", "rm "+testFile+"*").Run()

	db, err := NewSqliteDb(testFile)
	if err != nil {
		t.Fatal("Error opening db: ", err)
	}

	return db
}

func TestDbSqlite(t *testing.T) {
	db := newTestDb(t)
	defer db.Close()

	rootID := db.rootNodeID()

	if rootID == "" {
		t.Fatal("Root ID is blank: ", rootID)
	}

	rn, err := db.node(rootID)
	if err != nil {
		t.Fatal("Error getting root node: ", err)
	}

	if rn.ID == "" {
		t.Fatal("Root node ID is blank")
	}

	// modify a point and see if it changes
	err = db.nodePoints(rootID, data.Points{{Type: data.PointTypeDescription, Text: "root"}})
	if err != nil {
		t.Fatal(err)
	}

	rn, err = db.node(rootID)
	if err != nil {
		t.Fatal("Error getting root node: ", err)
	}

	if rn.Desc() != "root" {
		t.Fatal("Description should have been root, got: ", rn.Desc())
	}

	// send an old point and verify it does not change
	err = db.nodePoints(rootID, data.Points{{Time: time.Now().Add(-time.Hour),
		Type: data.PointTypeDescription, Text: "root with old time"}})
	if err != nil {
		t.Fatal(err)
	}

	rn, err = db.node(rootID)
	if err != nil {
		t.Fatal("Error getting root node: ", err)
	}

	if rn.Desc() != "root" {
		t.Fatal("Description should have stayed root, got: ", rn.Desc())
	}

	// verify default admin user got set
	children, err := db.children(rootID, "", false)
	if err != nil {
		t.Fatal("children error: ", err)
	}

	if len(children) < 1 {
		t.Fatal("did not return any children")
	}

	if children[0].Parent != rootID {
		t.Fatal("Parent not correct: ", children[0].Parent)
	}

	// test nodeEdge API
	adminID := children[0].ID

	adminNodes, err := db.nodeEdge(adminID, rootID)
	if err != nil {
		t.Fatal("Error getting admin nodes", err)
	}

	if len(adminNodes) < 1 {
		t.Fatal("did not return admin nodes")
	}

	if adminNodes[0].Type != data.NodeTypeUser {
		t.Fatal("nodeEdge did not return right node type for user")
	}

	adminNodes, err = db.nodeEdge(adminID, "none")
	if err != nil {
		t.Fatal("Error getting admin nodes", err)
	}

	if len(adminNodes) < 1 {
		t.Fatal("did not return admin nodes")
	}

	rootNodes, err := db.nodeEdge("root", "")
	if err != nil {
		t.Fatal("Error getting admin nodes", err)
	}

	if len(rootNodes) < 1 {
		t.Fatal("did not return root nodes")
	}

	if rootNodes[0].ID != rootID {
		t.Fatal("root node ID is not correct")
	}

	// test edge points
	err = db.edgePoints(adminID, rootID, data.Points{{Type: data.PointTypeRole, Text: data.PointValueRoleAdmin}})

	adminNodes, err = db.nodeEdge(adminID, rootID)
	if err != nil {
		t.Fatal("Error getting admin nodes", err)
	}

	p, ok := adminNodes[0].EdgePoints.Find(data.PointTypeRole, "")
	if !ok {
		t.Fatal("point not found")
	}
	if p.Text != data.PointValueRoleAdmin {
		t.Fatal("point does not have right value")
	}

	// try two children
	groupNodeID := uuid.New().String()

	err = db.nodePoints(groupNodeID, data.Points{{Type: data.PointTypeNodeType, Text: data.NodeTypeGroup}})
	if err != nil {
		t.Fatal("Error creating group node", err)
	}

	err = db.edgePoints(groupNodeID, rootID, data.Points{{Type: data.PointTypeTombstone, Value: 0}})
	if err != nil {
		t.Fatal("Error creating group edge", err)
	}

	// verify default admin user got set
	children, err = db.children(rootID, "", false)
	if err != nil {
		t.Fatal("children error: ", err)
	}

	if len(children) < 2 {
		t.Fatal("did not return 2 children")
	}

	// verify nodeEdge with "all" works
	start := time.Now()
	adminNodes, err = db.nodeEdge(adminID, "all")
	fmt.Println("nodeEdge time: ", time.Since(start))
	if err != nil {
		t.Fatal("Error getting admin nodes with all specified: ", err)
	}

	if adminNodes[0].Parent != rootID {
		t.Fatal("Parent ID is not correct")
	}

	if len(adminNodes) < 1 {
		t.Fatal("did not return admin nodes")
	}
}

func TestDbSqliteReopen(t *testing.T) {
	db := newTestDb(t)
	rootID := db.rootNodeID()
	db.Close()

	var err error
	db, err = NewSqliteDb(testFile)
	if err != nil {
		t.Fatal("Error opening db: ", err)
	}
	defer db.Close()

	if rootID != db.rootNodeID() {
		t.Fatal("Root node ID changed")
	}
}

func TestDbSqliteUserCheck(t *testing.T) {
	db := newTestDb(t)
	defer db.Close()

	nodes, err := db.userCheck("admin@admin.com", "admin")
	if err != nil {
		t.Fatal("userCheck returned error: ", err)
	}

	if len(nodes) < 1 {
		t.Fatal("userCheck did not return nodes")
	}
}
