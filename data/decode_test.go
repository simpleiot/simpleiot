package data

import (
	"reflect"
	"testing"
)

type TestType struct {
	Description string  `siot:"description"`
	Count       int     `siot:"count"`
	Value       float64 `siot:"value"`
	Role        string  `siotedge: "role"`
	Tombstone   bool    `siotedge: "tombstone"`
}

func TestDecode(t *testing.T) {

	in := NodeEdge{
		Type: "testType",
		Points: []Point{
			Point{Type: "Description", Text: "test type"},
			Point{Type: "count", Value: 120},
			Point{Type: "value", Value: 15.43},
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
		Role:        "admin",
		Tombstone:   true,
	}

	var out TestType

	err := Decode(in, out)

	if err != nil {
		t.Fatal("Error decoding: ", err)
	}

	if !reflect.DeepEqual(out, exp) {
		t.Errorf("Decode failed, exp: %v, got %v", exp, out)
	}

}
