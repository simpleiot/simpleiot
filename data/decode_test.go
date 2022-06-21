package data

import (
	"reflect"
	"testing"
)

type TestType struct {
	Description string  `point:"description"`
	Count       int     `point:"count"`
	Value       float64 `point:"value"`
	Value2      float32 `point:"value2"`
	Role        string  `edgepoint:"role"`
	Tombstone   bool    `edgepoint:"tombstone"`
}

func TestDecode(t *testing.T) {

	in := NodeEdge{
		Type: "testType",
		Points: []Point{
			Point{Type: "description", Text: "test type"},
			Point{Type: "count", Value: 120},
			Point{Type: "value", Value: 15.43},
			Point{Type: "value2", Value: 25.23},
		},
		EdgePoints: []Point{
			Point{Type: "role", Text: "admin"},
			Point{Type: "tombstone", Value: 1},
		},
	}

	exp := TestType{
		Description: "test type",
		Count:       120,
		Value:       15.43,
		Value2:      25.23,
		Role:        "admin",
		Tombstone:   true,
	}

	var out TestType

	err := Decode(in, &out)

	if err != nil {
		t.Fatal("Error decoding: ", err)
	}

	if !reflect.DeepEqual(out, exp) {
		t.Errorf("Decode failed, exp: %v, got %v", exp, out)
	}

}
