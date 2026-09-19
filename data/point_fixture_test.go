package data

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The JavaScript codec in frontend/lib/codec.mjs mirrors the point and
// node encodings here. These fixtures are what ties the two together: this
// test writes them when UPDATE_FIXTURES is set and otherwise checks that the
// Go encoder still produces the checked-in bytes, and the JavaScript tests
// decode the same bytes and compare against the JSON.

const fixtureDir = "../frontend/lib/testdata"

func fixturePoints() Points {
	t0 := time.Date(2026, 9, 1, 12, 30, 45, 123000000, time.UTC)
	pts := Points{
		NewPointFloat("value", "0", 12.5),
		NewPointInt("count", "3", 7),
		NewPointInt("big", "0", 1<<40),
		NewPointString("description", "0", "sensor one"),
		{Type: "empty", Key: "0"},
	}
	pts[3].Origin = "user-1"
	pts[3].Tombstone = 1
	pts = append(pts, Point{Type: "json", Key: "0", DataType: PointDataTypeJSON, Data: []byte(`{"a":1}`)})
	for i := range pts {
		pts[i].Time = t0.Add(time.Duration(i) * time.Second)
	}
	return pts
}

func fixtureNodes() Nodes {
	pts := fixturePoints()
	tomb := NewPointFloat("tombstone", "0", 0)
	tomb.Time = pts[0].Time
	return Nodes{
		{
			ID:         "node-1",
			Type:       "variable",
			Parent:     "group-1",
			Points:     pts[:3],
			EdgePoints: Points{tomb},
		},
		{
			ID:     "node-2",
			Type:   "group",
			Parent: "group-1",
			Points: pts[3:],
		},
	}
}

// pointFixture is the JSON shape the JavaScript decoder returns, which is
// the shape the web UI reads.
type pointFixture struct {
	Type      string  `json:"type"`
	Key       string  `json:"key"`
	Time      string  `json:"time"`
	DataType  int     `json:"dataType"`
	Value     float64 `json:"value"`
	Text      string  `json:"text"`
	Tombstone int     `json:"tombstone"`
	Origin    string  `json:"origin"`
}

func toFixture(p Point) pointFixture {
	return pointFixture{
		Type:      p.Type,
		Key:       p.Key,
		Time:      p.Time.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		DataType:  int(p.DataType),
		Value:     p.Val(),
		Text:      p.Txt(),
		Tombstone: p.Tombstone,
		Origin:    p.Origin,
	}
}

type nodeFixture struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Parent     string         `json:"parent"`
	Points     []pointFixture `json:"points"`
	EdgePoints []pointFixture `json:"edgePoints"`
}

func checkFixture(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join(fixtureDir, name)
	if os.Getenv("UPDATE_FIXTURES") != "" {
		if err := os.WriteFile(path, got, 0644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%v is missing; run with UPDATE_FIXTURES=1 to write it", path)
	}
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%v differs from the encoder; run with UPDATE_FIXTURES=1 if the change is intended", path)
	}
}

func TestPointFixtures(t *testing.T) {
	pts := fixturePoints()
	checkFixture(t, "points.bin", pts.Encode())

	var fx []pointFixture
	for _, p := range pts {
		fx = append(fx, toFixture(p))
	}
	j, _ := json.MarshalIndent(fx, "", "  ")
	checkFixture(t, "points.json", append(j, '\n'))

	nodes := fixtureNodes()
	checkFixture(t, "nodes.bin", EncodeNodes(nodes, nil))
	checkFixture(t, "nodes-error.bin", EncodeNodes(nil, errors.New("not in scope")))

	var nfx []nodeFixture
	for _, n := range nodes {
		f := nodeFixture{ID: n.ID, Type: n.Type, Parent: n.Parent, Points: []pointFixture{}, EdgePoints: []pointFixture{}}
		for _, p := range n.Points {
			f.Points = append(f.Points, toFixture(p))
		}
		for _, p := range n.EdgePoints {
			f.EdgePoints = append(f.EdgePoints, toFixture(p))
		}
		nfx = append(nfx, f)
	}
	j, _ = json.MarshalIndent(nfx, "", "  ")
	checkFixture(t, "nodes.json", append(j, '\n'))

	// the fixture round-trips through the Go decoder as well
	back, err := DecodePoints(pts.Encode())
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != len(pts) {
		t.Fatalf("decoded %v points, want %v", len(back), len(pts))
	}
	for i := range pts {
		if back[i].Type != pts[i].Type || back[i].Val() != pts[i].Val() || back[i].Txt() != pts[i].Txt() ||
			!back[i].Time.Equal(pts[i].Time) || back[i].Origin != pts[i].Origin {
			t.Errorf("point %v: got %v, want %v", i, back[i], pts[i])
		}
	}
}
