package data

import (
	"reflect"
	"testing"
)

type testType struct {
	Description string  `point:"description"`
	Count       int     `point:"count"`
	Value       float64 `point:"value"`
	Value2      float32 `point:"value2"`
	Role        string  `edgepoint:"role"`
	Tombstone   bool    `edgepoint:"tombstone"`
}

var nodeEdgeTest = NodeEdge{
	Type: "testType",
	Points: []Point{
		Point{Type: "description", Text: "test type"},
		Point{Type: "count", Value: 120},
		Point{Type: "value", Value: 15.43},
		Point{Type: "value2", Value: 10},
	},
	EdgePoints: []Point{
		Point{Type: "role", Text: "admin"},
		Point{Type: "tombstone", Value: 1},
	},
}

var testTypeData = testType{
	Description: "test type",
	Count:       120,
	Value:       15.43,
	Value2:      10,
	Role:        "admin",
	Tombstone:   true,
}

func TestDecode(t *testing.T) {
	var out testType

	err := Decode(nodeEdgeTest, &out)

	if err != nil {
		t.Fatal("Error decoding: ", err)
	}

	if !reflect.DeepEqual(out, testTypeData) {
		t.Errorf("Decode failed, exp: %v, got %v", testTypeData, out)
	}
}

func TestEncode(t *testing.T) {
	var out NodeEdge

	out, err := Encode(testTypeData)

	if err != nil {
		t.Fatal("Error encoding: ", err)
	}

	if !reflect.DeepEqual(out, nodeEdgeTest) {
		t.Errorf("Decode failed, exp: %v, got %v", nodeEdgeTest, out)
	}

}
