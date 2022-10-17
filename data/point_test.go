package data

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestPointsSort(t *testing.T) {
	now := time.Now()
	p1 := Point{Time: now}
	p2 := Point{Time: now.Add(time.Millisecond)}
	p3 := Point{Time: now.Add(time.Millisecond * 2)}

	exp := Points{p1, p2, p3}

	t1 := Points{p1, p2, p3}

	sort.Sort(t1)

	if !reflect.DeepEqual(t1, exp) {
		t.Errorf("t1 failed: t1: %v, exp: %v", t1, exp)
	}

	t2 := Points{p2, p3, p1}
	sort.Sort(t2)

	if !reflect.DeepEqual(t2, exp) {
		t.Errorf("t2 failed, t2: %v, exp: %v", t2, exp)
	}

	t3 := Points{p1, p2, p3}
	sort.Sort(t3)

	if !reflect.DeepEqual(t3, exp) {
		t.Errorf("t3 failed, t3: %v, exp: %v", t3, exp)
	}
}

func TestPointCRC(t *testing.T) {
	now := time.Now()
	p1 := Point{Type: "pointa", Time: now}
	p2 := Point{Type: "pointb", Time: now}

	fmt.Printf("p1 CRC: %0x\n", p1.CRC())
	fmt.Printf("p2 CRC: %0x\n", p2.CRC())

	if p1.CRC() == p2.CRC() {
		t.Error("CRC is weak")
	}
}
