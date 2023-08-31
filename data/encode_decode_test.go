package data

import (
	"reflect"
	"sort"
	"strconv"
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
	ID            string            `node:"id"`
	Parent        string            `node:"parent"`
	Description   string            `point:"description"`
	IPAddresses   []string          `point:"ipAddress"`
	Location      map[string]string `point:"location"`
	Sensors       map[string]int    `point:"sensor"`
	Nested        TestType          `point:"nested"`
	ScheduledDays [7]bool           `point:"scheduledDays"`
	TestValues    []int32           `edgepoint:"testValue"`
	Tombstone     bool              `edgepoint:"tombstone"`
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
	[7]bool{false, true, true, true, true, true, false},
	[]int32{314, 1024},
	true,
}

var nodeEdgeTestComplex = NodeEdge{
	ID:     "123",
	Parent: "456",
	Type:   "testTypeComplex",
	Points: []Point{
		{Type: "description", Text: "hi there"},
		{Type: "ipAddress", Key: "0", Text: "192.168.1.1"},
		{Type: "ipAddress", Key: "1", Text: "127.0.0.1"},
		{Type: "location", Key: "goodbye", Text: "cruel world"},
		{Type: "location", Key: "hello", Text: "world"},
		{Type: "nested", Key: "description", Text: "nested test type"},
		{Type: "nested", Key: "id", Text: "789"},
		{Type: "nested", Key: "parent", Text: "456"},
		{Type: "scheduledDays", Key: "0", Value: 0},
		{Type: "scheduledDays", Key: "1", Value: 1},
		{Type: "scheduledDays", Key: "2", Value: 1},
		{Type: "scheduledDays", Key: "3", Value: 1},
		{Type: "scheduledDays", Key: "4", Value: 1},
		{Type: "scheduledDays", Key: "5", Value: 1},
		{Type: "scheduledDays", Key: "6", Value: 0},
		{Type: "sensor", Key: "temp1", Value: 23},
		{Type: "sensor", Key: "temp2", Value: 40},
	},
	EdgePoints: []Point{
		{Type: "testValue", Key: "0", Value: 314},
		{Type: "testValue", Key: "1", Value: 1024},
		{Type: "tombstone", Value: 1},
	},
}

type testTypePointers struct {
	ID          string    `node:"id"`
	Description *string   `point:"description"`
	IPAddresses []string  `point:"ipAddress"`
	NullStruct  *TestType `point:"nullStruct"`
	NullValue   *float64  `point:"nullValue"`
	NullEdge    *int      `edgepoint:"nullEdge"`
	Value       *float32  `edgepoint:"value"`
}

var testTypePointersNodeEdge = NodeEdge{
	ID:   "nodeID",
	Type: "testTypePointers",
	Points: []Point{
		{Type: "description", Text: "testing 1, 2, 3"},
		{Type: "nullStruct", Key: "description", Tombstone: 1},
		{Type: "nullStruct", Key: "id", Tombstone: 1},
		{Type: "nullStruct", Key: "parent", Tombstone: 1},
		{Type: "nullValue", Tombstone: 1},
	},
	EdgePoints: []Point{
		{Type: "nullEdge", Tombstone: 1},
		{Type: "value", Value: 42},
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

func TestDecodeTombstonePoint(t *testing.T) {
	var ne = NodeEdge{
		Points: []Point{
			{Type: "ipAddress", Key: "0", Text: "192.168.1.1"},
			{Type: "ipAddress", Key: "1", Text: "127.0.0.1"},
			{Type: "ipAddress", Key: "2", Text: "127.0.0.2", Tombstone: 1},
			{Type: "location", Key: "goodbye", Text: "cruel world"},
			{Type: "location", Key: "hello", Text: "world"},
			{Type: "location", Key: "del", Text: "deleted entry", Tombstone: 1},
			{Type: "nested", Key: "fake", Text: "not a real field", Tombstone: 1},
		},
	}

	var out testTypeComplex
	out.Nested.Description = "decode should not change this value"
	err := Decode(NodeEdgeChildren{ne, nil}, &out)

	exp := testTypeComplex{
		Nested: TestType{
			Description: "decode should not change this value",
		},
		IPAddresses: []string{"192.168.1.1", "127.0.0.1"},
		Location: map[string]string{
			"hello":   "world",
			"goodbye": "cruel world",
		},
	}

	if err != nil {
		t.Fatal("Error decoding: ", err)
	}

	if !reflect.DeepEqual(out, exp) {
		t.Errorf("Decode failed, exp: %v, got %v", exp, out)
	}
}

func TestDecodeAllTombstonePointArray(t *testing.T) {
	var ne = NodeEdge{
		Points: []Point{
			{Type: "ipAddress", Key: "0", Text: "192.168.1.1", Tombstone: 1},
			{Type: "ipAddress", Key: "1", Text: "127.0.0.1", Tombstone: 1},
			{Type: "ipAddress", Key: "2", Text: "127.0.0.2", Tombstone: 1},
		},
	}

	var out testTypeComplex
	err := Decode(NodeEdgeChildren{ne, nil}, &out)

	if err != nil {
		t.Fatal("Error decoding: ", err)
	}

	if len(out.IPAddresses) > 0 {
		t.Error("Expected 0 IP address, got: ", len(out.IPAddresses))
	}
}

func TestEncodePointers(t *testing.T) {
	str := "testing 1, 2, 3"
	value := float32(42)
	ne, err := Encode(testTypePointers{
		ID:          "nodeID",
		Description: &str,
		Value:       &value,
	})

	if err != nil {
		t.Fatal("encode failed:", err)
	}
	sortPoints(ne.Points, ne.EdgePoints)

	if !reflect.DeepEqual(ne, testTypePointersNodeEdge) {
		t.Errorf("Decode failed, exp: %v, got %v", testTypePointersNodeEdge, ne)
	}
}

func TestDecodePointers(t *testing.T) {
	desc := "Test description"
	nullValue := 85.7
	out := testTypePointers{
		ID:          "123",
		Description: &desc,
		IPAddresses: []string{"127.0.0.1"},
		NullStruct: &TestType{
			Description: "hello there",
		},
		NullValue: &nullValue,
	}
	err := Decode(NodeEdgeChildren{testTypePointersNodeEdge, nil}, &out)

	desc = "testing 1, 2, 3"
	value := float32(42)
	exp := testTypePointers{
		ID:          "nodeID",
		Description: &desc,
		IPAddresses: []string{"127.0.0.1"}, // unchanged
		NullStruct:  nil,                   // all fields are tombstone points
		NullValue:   nil,                   // tombstone point
		NullEdge:    nil,                   // tombstone point
		Value:       &value,
	}

	if err != nil {
		t.Fatal("Error decoding: ", err)
	}

	if !reflect.DeepEqual(out, exp) {
		t.Errorf("Decode failed, exp: %v, got %v", exp, out)
	}
}

func TestDiffPoints(t *testing.T) {
	before := testType{
		ID:          "123",
		Parent:      "456",
		Description: "test type",
		Count:       120,
		Value:       15.43,
		Value2:      10,
		Role:        "admin",
		Tombstone:   true,
	}
	after := testType{
		ID:          "0123",
		Parent:      "0456",
		Description: "description changed",
		Count:       110,
		Value:       15.43, // unchanged
		Value2:      10000000,
		Role:        "user",
		Tombstone:   false,
	}
	p, err := DiffPoints(before, after)
	if err != nil {
		t.Fatal("diff error:", err)
	}
	if len(p) != 3 {
		t.Fatalf("expected 3 points; got %v", len(p))
	}
	if p[0].Value != 0 ||
		p[0].Text != "description changed" ||
		p[0].Type != "description" ||
		p[0].Tombstone != 0 {
		t.Errorf("generated point invalid; got %v", p[0])
	}
	if p[1].Value != 110 ||
		p[1].Text != "" ||
		p[1].Type != "count" ||
		p[1].Tombstone != 0 {
		t.Errorf("generated point invalid; got %v", p[1])
	}
	if p[2].Value != 10000000 ||
		p[2].Text != "" ||
		p[2].Type != "value2" ||
		p[2].Tombstone != 0 {
		t.Errorf("generated point invalid; got %v", p[2])
	}
}

func TestDiffPointsComplex(t *testing.T) {
	// type testTypeComplex struct {
	// 	ID          string            `node:"id"`
	// 	Parent      string            `node:"parent"`
	// 	Description string            `point:"description"`
	// 	IPAddresses []string          `point:"ipAddress"`
	// 	Location    map[string]string `point:"location"`
	// 	Sensors     map[string]int    `point:"sensor"`
	// 	Nested      TestType          `point:"nested"`
	// 	TestValues  []int32           `edgepoint:"testValue"`
	// 	Tombstone   bool              `edgepoint:"tombstone"`
	// }
	before := testTypeComplex{"123", "456",
		"hi there",
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
		[7]bool{false, true, true, true, true, true, false},
		[]int32{314, 1024},
		true,
	}
	after := testTypeComplex{"0123", "0456",
		"hi there",                // unchanged
		[]string{"192.168.1.100"}, // index 0 updated; 1 deleted
		map[string]string{
			"hello": "world!!!", // hello updated; goodbye deleted
			"foo":   "bar",      // foo added
		},
		map[string]int{
			"temp1": 23,
			"temp2": 40, // unchanged
		},
		TestType{"789", "456", "nested test type desc changed"},
		[7]bool{false, true, true, false, true, true, false},
		// ignore edgepoints
		[]int32{314, 1000, 2048, 4096},
		false,
	}
	p, err := DiffPoints(before, after)
	if err != nil {
		t.Fatal("diff error:", err)
	}
	sortPoints(p)
	// log.Println(p)
	if len(p) != 7 {
		t.Fatalf("expected 7 points; got %v", len(p))
	}
	if p[0].Value != 0 ||
		p[0].Text != "192.168.1.100" ||
		p[0].Key != "0" ||
		p[0].Type != "ipAddress" ||
		p[0].Tombstone != 0 {
		t.Errorf("generated point invalid; got %v", p[0])
	}
	if p[1].Value != 0 ||
		p[1].Text != "" ||
		p[1].Key != "1" ||
		p[1].Type != "ipAddress" ||
		p[1].Tombstone != 1 {
		t.Errorf("generated point invalid; got %v", p[1])
	}
	if p[2].Value != 0 ||
		p[2].Text != "bar" ||
		p[2].Key != "foo" ||
		p[2].Type != "location" ||
		p[2].Tombstone != 0 {
		t.Errorf("generated point invalid; got %v", p[3])
	}
	if p[3].Value != 0 ||
		p[3].Text != "" ||
		p[3].Key != "goodbye" ||
		p[3].Type != "location" ||
		p[3].Tombstone != 1 {
		t.Errorf("generated point invalid; got %v", p[4])
	}
	if p[4].Value != 0 ||
		p[4].Text != "world!!!" ||
		p[4].Key != "hello" ||
		p[4].Type != "location" ||
		p[4].Tombstone != 0 {
		t.Errorf("generated point invalid; got %v", p[2])
	}
	if p[5].Value != 0 ||
		p[5].Text != "nested test type desc changed" ||
		p[5].Key != "description" ||
		p[5].Type != "nested" ||
		p[5].Tombstone != 0 {
		t.Errorf("generated point invalid; got %v", p[5])
	}
	if p[6].Value != 0 ||
		p[6].Text != "" ||
		p[6].Key != "3" ||
		p[6].Type != "scheduledDays" ||
		p[6].Tombstone != 0 {
		t.Errorf("generated point invalid; got %v", p[5])
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

	// try to sort Key as int first (array), then as text

	iInt, iErr := strconv.Atoi(sp[i].Key)
	jInt, jErr := strconv.Atoi(sp[j].Key)

	if iErr == nil && jErr == nil {
		// we have ints, so do int sort
		if iInt < jInt {
			return true
		}

		if iInt > jInt {
			return false
		}
	} else {
		if sp[i].Key < sp[j].Key {
			return true
		}
		if sp[i].Key > sp[j].Key {
			return false
		}
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
