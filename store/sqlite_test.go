package store

import (
	"os"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

func TestDbSqlite(t *testing.T) {
	testFile := "test.sqlite"
	os.Remove(testFile)

	db, err := NewSqliteDb("./", testFile)
	if err != nil {
		t.Fatal("Error opening db: ", err)
	}
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
	children, err := db.children(rootID)
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

	adminNodes, err := db.nodeEdge(rootID, adminID)
	if err != nil {
		t.Fatal("Error getting admin nodes", err)
	}

	if len(adminNodes) < 1 {
		t.Fatal("did not return admin nodes")
	}

	adminNodes, err = db.nodeEdge(rootID, "none")
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

}
