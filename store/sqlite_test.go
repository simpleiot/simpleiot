package store

import (
	"os"
	"testing"
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
}
