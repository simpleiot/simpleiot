package db

import (
	"testing"
)

func TestNodeDelete(t *testing.T) {
	db, err := NewDb(StoreTypeMemory, "")
	if err != nil {
		t.Error(err)
	}

	_ = db

	// FIXME move this test to NATS api

	/*

		node1 := data.NodeEdge{
			Type:   data.NodeTypeModbus,
			Parent: db.RootNodeID(),
		}

		node1ID, err := db.NodeInsertEdge(node1)
		if err != nil {
			t.Error(err)
		}

		node2 := data.NodeEdge{
			Type:   data.NodeTypeModbusIO,
			Parent: node1ID,
		}

		node2ID, err := db.NodeInsertEdge(node2)
		if err != nil {
			t.Error(err)
		}

		err = db.NodeDelete(node1ID, db.RootNodeID())
		if err != nil {
			t.Error(err)
		}

		_, err = db.Node(node1ID)
		if err == nil {
			t.Error("found node1, should have been deleted")
		}

		_, err = db.Node(node2ID)
		if err == nil {
			t.Error("found node2, should have been deleted")
		}
	*/
}
