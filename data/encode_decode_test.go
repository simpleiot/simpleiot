package data

import (
	"reflect"
	"sort"
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

type testTypeComplex struct {
	ID          string            `node:"id"`
	Parent      string            `node:"parent"`
	Description string            `point:"description"`
	IPAddresses []string          `point:"ipAddress"`
	Location    map[string]string `point:"location"`
	Sensors     map[string]int    `point:"sensor"`
	Nested      TestType          `point:"nested"`
	TestValues  []int32           `edgepoint:"testValue"`
	Tombstone   bool              `edgepoint:"tombstone"`
}

var testTypeComplexData = testTypeComplex{"123", "456", "hi there",
	[]string{"192.168.1.1", "127.0.0.1"},
	map[string]string{
		"hello":   "world",
		"goodbye": "cruel world",
	},
	map[string]int{
		"temp1": 23,
		"temp2": 40,
	},
	TestType{"789", "456", "nested test type"},
	[]int32{314, 1024},
	true,
}

var nodeEdgeTestComplex = NodeEdge{
	ID:     "123",
	Parent: "456",
	Type:   "testTypeComplex",
	Points: []Point{
		{Type: "description", Text: "hi there"},
		{Type: "ipAddress", Index: 0, Text: "192.168.1.1"},
		{Type: "ipAddress", Index: 1, Text: "127.0.0.1"},
		{Type: "location", Key: "goodbye", Text: "cruel world"},
		{Type: "location", Key: "hello", Text: "world"},
		{Type: "nested", Key: "description", Text: "nested test type"},
		{Type: "nested", Key: "id", Text: "789"},
		{Type: "nested", Key: "parent", Text: "456"},
		{Type: "sensor", Key: "temp1", Value: 23},
		{Type: "sensor", Key: "temp2", Value: 40},
	},
	EdgePoints: []Point{
		{Type: "testValue", Index: 0, Value: 314},
		{Type: "testValue", Index: 1, Value: 1024},
		{Type: "tombstone", Value: 1},
	},
}

func TestEncodeComplex(t *testing.T) {
	ne, err := Encode(testTypeComplexData)

	if err != nil {
		t.Fatal("encode failed:", err)
	}
	sortPoints(ne.Points, ne.EdgePoints)

	if !reflect.DeepEqual(ne, nodeEdgeTestComplex) {
		t.Errorf("Decode failed, exp: %v, got %v", nodeEdgeTestComplex, ne)
	}
}

func TestDecodeComplex(t *testing.T) {
	var out testTypeComplex

	err := Decode(NodeEdgeChildren{nodeEdgeTestComplex, nil}, &out)

	if err != nil {
		t.Fatal("Error decoding: ", err)
	}

	if !reflect.DeepEqual(out, testTypeComplexData) {
		t.Errorf("Decode failed, exp: %v, got %v", testTypeComplexData, out)
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

type SortablePoints []Point

func (sp SortablePoints) Len() int {
	return len(sp)
}

// Sort by type, key, index, and then time
func (sp SortablePoints) Less(i, j int) bool {
	if sp[i].Type < sp[j].Type {
		return true
	}
	if sp[i].Type > sp[j].Type {
		return false
	}

	if sp[i].Key < sp[j].Key {
		return true
	}
	if sp[i].Key > sp[j].Key {
		return false
	}

	if sp[i].Index < sp[j].Index {
		return true
	}
	if sp[i].Index > sp[j].Index {
		return false
	}

	return sp[i].Time.Before(sp[j].Time)
}

func (sp SortablePoints) Swap(i, j int) {
	sp[i], sp[j] = sp[j], sp[i]
}

func sortPoints(slices ...[]Point) {
	for _, pts := range slices {
		sort.Sort(SortablePoints(pts))
	}
}
