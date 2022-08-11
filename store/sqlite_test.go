package store

import "testing"

func TestDbSqlite(t *testing.T) {
	db, err := NewSqliteDb("./", TypeFile)
	if err != nil {
		t.Fatal("Error opening db: ", err)
	}

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
