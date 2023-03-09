package data

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
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

func TestDecodeSerialHrPayload(t *testing.T) {
	var buf bytes.Buffer

	b := make([]byte, 16)
	copy(b, []byte("voltage"))
	buf.Write(b)

	b = make([]byte, 16)
	copy(b, []byte("AX"))
	buf.Write(b)

	start := time.Now()
	startNs := start.UnixNano()
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(startNs))
	buf.Write(b)

	samp := (time.Millisecond * 50).Nanoseconds()
	b = make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(samp))
	buf.Write(b)

	binary.LittleEndian.PutUint32(b, math.Float32bits(10.5))
	buf.Write(b)

	binary.LittleEndian.PutUint32(b, math.Float32bits(1000.23))
	buf.Write(b)

	var pts Points

	err := DecodeSerialHrPayload(buf.Bytes(), func(p Point) {
		pts = append(pts, p)
	})

	if err != nil {
		t.Fatal("Error decoding: ", err)
	}

	exp := Points{
		{Time: start, Type: "voltage", Key: "AX", Value: 10.5},
		{Time: start.Add(time.Millisecond * 50), Type: "voltage", Key: "AX", Value: 1000.23},
	}

	if len(exp) != len(pts) {
		t.Fatal("Did not get the exp # points")
	}

	for i, e := range exp {
		p := pts[i]

		if p.Time.Sub(e.Time).Abs() > time.Nanosecond {
			t.Error("Time not equal")
		}
		if p.Type != e.Type {
			t.Error("Type not equal")
		}
		if p.Key != e.Key {
			t.Error("Key not equal")
		}
	}
}
