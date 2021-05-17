package data

import (
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestPointsSort(t *testing.T) {
	p1 := Point{Time: time.Now()}
	p2 := Point{Time: time.Now()}
	p3 := Point{Time: time.Now()}

	exp := Points{p1, p2, p3}

	t1 := Points{p1, p2, p3}

	sort.Sort(t1)

	if !reflect.DeepEqual(t1, exp) {
		t.Error("t1 failed")
	}

	t2 := Points{p2, p3, p1}
	sort.Sort(t2)

	if !reflect.DeepEqual(t2, exp) {
		t.Error("t2 failed")
	}

	t3 := Points{p1, p2, p3}
	sort.Sort(t3)

	if !reflect.DeepEqual(t3, exp) {
		t.Error("t3 failed")
	}
}
