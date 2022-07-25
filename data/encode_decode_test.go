package data

import (
	"reflect"
	"testing"
)

type testType struct {
	ID          string  `node:"id"`
	Parent      string  `node:"parent"`
	Description string  `point:"description"`
	Count       int     `point:"count"`
	Value       float64 `point:"value"`
	Value2      float32 `point:"value2"`
	Role        string  `edgepoint:"role"`
	Tombstone   bool    `edgepoint:"tombstone"`
}

var nodeEdgeTest = NodeEdge{
	ID:     "123",
	Parent: "456",
	Type:   "testType",
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
	ID:          "123",
	Parent:      "456",
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

type testType2 struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description int    `point:"description"`
}

func TestDecodeTypeMismatch(t *testing.T) {
	var out testType2

	err := Decode(nodeEdgeTest, &out)

	if err != nil {
		t.Fatal("Error decoding type mismatch test: ", err)
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

func TestMergePoints(t *testing.T) {
	out := testTypeData

	modifiedDescription := "test type modified"

	mods := []Point{
		{Type: "description", Text: modifiedDescription},
	}

	err := MergePoints(mods, &out)

	if err != nil {
		t.Fatal("Merge error: ", err)
	}

	if out.Description != modifiedDescription {
		t.Errorf("Description not modified, exp: %v, got: %v", modifiedDescription,
			out.Description)
	}
}

func TestMergeEdgePoints(t *testing.T) {
	out := testTypeData

	modifiedRole := "user"

	mods := []Point{
		{Type: "role", Text: modifiedRole},
	}

	err := MergeEdgePoints(mods, &out)

	if err != nil {
		t.Fatal("Merge error: ", err)
	}

	if out.Role != modifiedRole {
		t.Errorf("role not modified, exp: %v, got: %v", modifiedRole,
			out.Role)
	}
}

type TestType struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
}

func TestEncodeCase(t *testing.T) {
	test := TestType{"123", "456", "hi there"}

	ne, err := Encode(test)

	if err != nil {
		t.Fatal("encode failed: ", err)
	}

	if ne.Type != "testType" {
		t.Error("expected testType, got: ", ne.Type)
	}
}
