package data

import (
	"testing"
)

func TestMergePoints(t *testing.T) {
	out := testTypeData

	modifiedDescription := "test type modified"

	mods := []Point{
		{Type: "description", Text: modifiedDescription},
	}

	err := MergePoints(out.ID, mods, &out)

	if err != nil {
		t.Fatal("Merge error: ", err)
	}

	if out.Description != modifiedDescription {
		t.Errorf("Description not modified, exp: %v, got: %v", modifiedDescription,
			out.Description)
	}

	// make sure other points did not get reset
}

func TestMergeEdgePoints(t *testing.T) {
	out := testTypeData

	modifiedRole := "user"

	mods := []Point{
		{Type: "role", Text: modifiedRole},
	}

	err := MergeEdgePoints(out.ID, out.Parent, mods, &out)

	if err != nil {
		t.Fatal("Merge error: ", err)
	}

	if out.Role != modifiedRole {
		t.Errorf("role not modified, exp: %v, got: %v", modifiedRole,
			out.Role)
	}
}

func TestMergeChildPoints(t *testing.T) {
	testData := testX{
		ID:          "ID-testX",
		Parent:      "ID-parent",
		Description: "test X node",
		TestYs: []testY{
			{ID: "ID-testY",
				Parent:      "ID-testX",
				Description: "test Y node",
				Count:       3,
				Role:        "",
				TestZs: []testZ{
					{
						ID:          "ID-testZ",
						Parent:      "ID-testY",
						Description: "test Z node",
						Count:       23,
						Role:        "peon",
					},
				},
			},
		},
	}

	modifiedDescription := "test type modified"

	mods := []Point{
		{Type: "description", Text: modifiedDescription},
	}

	err := MergePoints("ID-testY", mods, &testData)

	if err != nil {
		t.Fatal("Merge error: ", err)
	}

	if testData.TestYs[0].Description != modifiedDescription {
		t.Errorf("Description not modified, exp: %v, got: %v", modifiedDescription,
			testData.TestYs[0].Description)
	}

	// make sure other points did not get reset
	if testData.TestYs[0].Count != 3 {
		t.Errorf("Merge reset other data")
	}

	if testData.Description != "test X node" {
		t.Errorf("Top level node description modified when it should not have")
	}

	// modify description of Z point
	modifiedDescription = "test Z type modified"

	mods = []Point{
		{Type: "description", Text: modifiedDescription},
	}

	err = MergePoints("ID-testZ", mods, &testData)
	if err != nil {
		t.Fatal("Merge error: ", err)
	}

	if testData.TestYs[0].TestZs[0].Description != modifiedDescription {
		t.Errorf("Description not modified, exp: %v, got: %v", modifiedDescription,
			testData.TestYs[0].TestZs[0].Description)
	}

	// Test edge modifications
	modifiedRole := "yrole"

	mods = []Point{
		{Type: "role", Text: modifiedRole},
	}

	err = MergeEdgePoints("ID-testZ", "ID-testY", mods, &testData)
	if err != nil {
		t.Fatal("Merge error: ", err)
	}

	if testData.TestYs[0].TestZs[0].Role != modifiedRole {
		t.Errorf("Role not modified, exp: %v, got: %v", modifiedRole,
			testData.TestYs[0].TestZs[0].Role)
	}
}

func TestMergeComplex(t *testing.T) {
	td := testTypeComplex{
		ID:          "ID-TC",
		Parent:      "456",
		Description: "hi there",
		IPAddresses: []string{"192.168.1.1", "127.0.0.1"},
		Location: map[string]string{
			"hello":   "world",
			"goodbye": "cruel world",
		},
		Sensors: map[string]int{
			"temp1": 23,
			"temp2": 40,
		},
		Nested:     TestType{"789", "456", "nested test type"},
		TestValues: []int32{314, 1024},
		Tombstone:  false,
	}

	p := Points{{Type: "location", Key: "hello", Text: "Siot"}}

	err := MergePoints("ID-TC", p, &td)

	if err != nil {
		t.Fatal("Error merging points to complex struct: ", err)
	}

	if td.Location["hello"] != "Siot" {
		t.Fatal("Map not modified to Siot")
	}

	ep := Points{{Type: "testValue", Value: 123}}

	err = MergeEdgePoints("ID-TC", "456", ep, &td)

	if err != nil {
		t.Fatal("Error merging points to complex struct: ", err)
	}

	if td.TestValues[0] != 123 {
		t.Fatal("edge point array not modified")
	}

	// delete points in array
	p = Points{
		{Type: "ipAddress", Key: "0", Tombstone: 1},
		{Type: "ipAddress", Key: "1", Tombstone: 1},
	}

	err = MergePoints("ID-TC", p, &td)
	if err != nil {
		t.Fatal("Error deleting array entries: ", err)
	}

	if len(td.IPAddresses) > 0 {
		t.Fatal("Expected 0 IP addresses, got: ", len(td.IPAddresses))
	}

	// delete a map entry
	p = Points{{Type: "sensor", Key: "temp1", Tombstone: 1}}

	err = MergePoints("ID-TC", p, &td)
	if err != nil {
		t.Fatal("Error deleting key entry: ", err)
	}

	_, ok := td.Sensors["temp1"]

	if ok {
		t.Fatal("Expected temp key to be deleted, got: ", td.Sensors)
	}
}
