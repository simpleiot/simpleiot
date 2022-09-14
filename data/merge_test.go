package data

import "testing"

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

	err := MergeEdgePoints(mods, &out)

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
}
