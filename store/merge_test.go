package store

import (
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

func TestTipWins(t *testing.T) {
	t0 := time.Now()
	t1 := t0.Add(time.Second)

	tests := []struct {
		name                string
		curTime, inTime     time.Time
		curOrigin, inOrigin string
		want                bool
	}{
		{"newer wins", t0, t1, "A", "A", true},
		{"older loses", t1, t0, "A", "A", false},
		{"equal time same origin is a no-op", t0, t0, "A", "A", false},
		{"equal time greater origin wins", t0, t0, "A", "B", true},
		{"equal time lesser origin loses", t0, t0, "B", "A", false},
		{"equal time known origin beats unknown", t0, t0, "", "A", true},
	}

	for _, tt := range tests {
		got := tipWins(tt.curTime, tt.curOrigin, tt.inTime, tt.inOrigin)
		if got != tt.want {
			t.Errorf("%v: tipWins = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// TestTipWinsConverges verifies the property the rule exists for: two
// instances merging the same two writes in either order agree on the
// winner.
func TestTipWinsConverges(t *testing.T) {
	now := time.Now()
	writes := []struct {
		t      time.Time
		origin string
	}{
		{now, "A"},
		{now, "B"},
		{now.Add(-time.Second), "C"},
	}

	// apply in order 0,1,2 and in order 2,1,0; both must land on the
	// same tip
	apply := func(order []int) (time.Time, string) {
		var tipT time.Time
		tipO := ""
		first := true
		for _, i := range order {
			w := writes[i]
			if first || tipWins(tipT, tipO, w.t, w.origin) {
				tipT, tipO = w.t, w.origin
				first = false
			}
		}
		return tipT, tipO
	}

	tA, oA := apply([]int{0, 1, 2})
	tB, oB := apply([]int{2, 1, 0})

	if !tA.Equal(tB) || oA != oB {
		t.Errorf("merge order changed the winner: (%v,%v) vs (%v,%v)",
			tA, oA, tB, oB)
	}
	if oA != "B" {
		t.Errorf("winner = %v, want B (equal time, greater origin)", oA)
	}
}

func TestMergeEdgePointsCrossOrigin(t *testing.T) {
	ec := NewEdgeCache()
	now := time.Now()

	mkPoint := func(typ, txt string, ts time.Time) data.Point {
		p := data.NewPointString(typ, "0", txt)
		p.Time = ts
		return p
	}

	// origin X writes the edge with a nodeType and description
	ec.MergeEdgePoints("P", "C", "variable", "X", data.Points{
		mkPoint(data.PointTypeNodeType, "variable", now),
		mkPoint(data.PointTypeDescription, "from X", now),
	})

	// origin R writes a newer description on the same edge
	ec.MergeEdgePoints("P", "C", "", "R", data.Points{
		mkPoint(data.PointTypeDescription, "from R", now.Add(time.Second)),
	})

	e, ok := ec.Get("P", "C")
	if !ok {
		t.Fatal("edge not found")
	}
	if e.Type != "variable" {
		t.Error("nodeType lost in merge:", e.Type)
	}
	d, _ := e.Points.Text(data.PointTypeDescription, "")
	if d != "from R" {
		t.Error("newer point did not win, got:", d)
	}

	// an older write from X must not replace R's newer tip
	ec.MergeEdgePoints("P", "C", "", "X", data.Points{
		mkPoint(data.PointTypeDescription, "stale from X", now),
	})
	e, _ = ec.Get("P", "C")
	d, _ = e.Points.Text(data.PointTypeDescription, "")
	if d != "from R" {
		t.Error("older point replaced newer tip, got:", d)
	}

	// equal-timestamp writes from two origins resolve to the greater
	// origin regardless of arrival order
	ts := now.Add(time.Minute)
	ec.MergeEdgePoints("P", "C", "", "X", data.Points{
		mkPoint(data.PointTypeDescription, "tie from X", ts),
	})
	ec.MergeEdgePoints("P", "C", "", "R", data.Points{
		mkPoint(data.PointTypeDescription, "tie from R", ts),
	})
	e, _ = ec.Get("P", "C")
	d, _ = e.Points.Text(data.PointTypeDescription, "")
	if d != "tie from X" {
		t.Error("origin tie-break failed, got:", d)
	}
}
