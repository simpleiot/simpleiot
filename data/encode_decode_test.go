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
		{Type: "description", Text: "test type"},
		{Type: "count", Value: 120},
		{Type: "value", Value: 15.43},
		{Type: "value2", Value: 10},
	},
	EdgePoints: []Point{
		{Type: "role", Text: "admin"},
		{Type: "tombstone", Value: 1},
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

	err := Decode(NodeEdgeChildren{nodeEdgeTest, nil}, &out)

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

	err := Decode(NodeEdgeChildren{nodeEdgeTest, nil}, &out)

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

type testX struct {
	ID          string  `node:"id"`
	Parent      string  `node:"parent"`
	Description string  `point:"description"`
	TestYs      []testY `child:"testY"`
}

type testY struct {
	ID          string  `node:"id"`
	Parent      string  `node:"parent"`
	Description string  `point:"description"`
	Count       int     `point:"count"`
	Role        string  `edgepoint:"role"`
	TestZs      []testZ `child:"testZ"`
	TestYs      []testY `child:"testY"`
}

type testZ struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Count       int    `point:"count"`
	Role        string `edgepoint:"role"`
}

func TestDecodeWithChildren(t *testing.T) {
	nX := NodeEdgeChildren{
		NodeEdge: NodeEdge{
			ID:     "123",
			Parent: "456",
			Type:   "testX",
			Points: []Point{
				{Type: "description", Text: "test X type"},
			},
			EdgePoints: []Point{
				{Type: "role", Text: "admin"},
				{Type: "tombstone", Value: 1},
			},
		},
		Children: []NodeEdgeChildren{
			{NodeEdge{
				ID:     "abc",
				Parent: "123",
				Type:   "testY",
				Points: []Point{
					{Type: "description", Text: "test Y1"},
				},
				EdgePoints: []Point{
					{Type: "role", Text: "user"},
					{Type: "tombstone", Value: 1},
				},
			},
				[]NodeEdgeChildren{
					{NodeEdge{
						ID:     "jkl",
						Parent: "abc",
						Type:   "testY",
						Points: []Point{
							{Type: "description", Text: "test Y2"},
						},
						EdgePoints: []Point{
							{Type: "role", Text: "user"},
							{Type: "tombstone", Value: 1},
						},
					}, nil},
					{NodeEdge{
						ID:     "mno",
						Parent: "abc",
						Type:   "testZ",
						Points: []Point{
							{Type: "description", Text: "test Z1"},
						},
						EdgePoints: []Point{
							{Type: "role", Text: "user"},
							{Type: "tombstone", Value: 1},
						},
					}, nil},
				},
			},
		},
	}

	var out testX

	err := Decode(nX, &out)

	if err != nil {
		t.Fatal("Error decoding: ", err)
	}

	if out.ID != "123" {
		t.Fatal("Decode failed, wrong ID")
	}

	if len(out.TestYs) < 1 {
		t.Fatal("No TestYs")
	}

	if out.TestYs[0].ID != "abc" {
		t.Fatal("Decode failed, wrong ID for TestYs[0]")
	}

	if len(out.TestYs[0].TestYs) < 1 {
		t.Fatal("No TestYs.TestYs")
	}

	if len(out.TestYs[0].TestZs) < 1 {
		t.Fatal("No TestYs.TestZs")
	}

}
