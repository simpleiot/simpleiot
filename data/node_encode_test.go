package data

import (
	"errors"
	"testing"
	"time"
)

func TestEncodeDecodeNodes(t *testing.T) {
	now := time.Now().Truncate(0)
	nodes := Nodes{
		{
			ID:     "n1",
			Type:   NodeTypeDevice,
			Parent: "root",
			Points: Points{
				{Type: "value", Key: "0", Time: now, DataType: PointDataTypeFloat,
					Data: NewPointFloat("value", "0", 1.5).Data, Origin: "o"},
				{Type: "description", Time: now, DataType: PointDataTypeString,
					Data: []byte("pump")},
			},
			EdgePoints: Points{
				{Type: PointTypeTombstone, Time: now, DataType: PointDataTypeInt,
					Data: NewPointInt(PointTypeTombstone, "", 0).Data, Tombstone: 1},
			},
		},
		{ID: "n2", Type: NodeTypeGroup, Parent: "n1"},
	}

	got, err := DecodeNodes(EncodeNodes(nodes, nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(nodes) {
		t.Fatalf("got %d nodes, want %d", len(got), len(nodes))
	}
	for i := range nodes {
		want := nodes[i]
		g := got[i]
		if g.ID != want.ID || g.Type != want.Type || g.Parent != want.Parent {
			t.Errorf("node %d header mismatch: got %+v want %+v", i, g, want)
		}
		if len(g.Points) != len(want.Points) || len(g.EdgePoints) != len(want.EdgePoints) {
			t.Fatalf("node %d point count mismatch", i)
		}
		for j := range want.Points {
			if g.Points[j].String() != want.Points[j].String() ||
				!g.Points[j].Time.Equal(want.Points[j].Time) {
				t.Errorf("node %d point %d: got %v want %v", i, j, g.Points[j], want.Points[j])
			}
		}
		for j := range want.EdgePoints {
			if g.EdgePoints[j].Tombstone != want.EdgePoints[j].Tombstone {
				t.Errorf("node %d edge point %d tombstone mismatch", i, j)
			}
		}
	}
}

func TestDecodeNodesError(t *testing.T) {
	_, err := DecodeNodes(EncodeNodes(nil, ErrDocumentNotFound))
	if !errors.Is(err, ErrDocumentNotFound) {
		t.Fatalf("want ErrDocumentNotFound, got %v", err)
	}

	_, err = DecodeNodes(EncodeNodes(nil, errors.New("boom")))
	if err == nil || err.Error() != "boom" {
		t.Fatalf("want boom, got %v", err)
	}

	got, err := DecodeNodes(nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("empty payload: got %v, %v", got, err)
	}

	if _, err := DecodeNodes([]byte{9, 0, 0}); err == nil {
		t.Fatal("want error for unknown frame version")
	}

	if _, err := DecodeNodes(EncodeNodes(nil, nil)[:3]); err == nil {
		t.Fatal("want error for truncated frame")
	}
}
