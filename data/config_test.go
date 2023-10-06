package data

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/goccy/go-yaml"
)

var configTestNode = NodeEdgeChildren{
	NodeEdge: NodeEdge{
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
	},
}

var configTestYAML = `
nodes:
  - type: testType
    parent: 456
    id: 123
    points:
      - {type: description, text: "test type"}
      - {type: count, value: 120}
      - {type: value, value: 15.43}
      - {type: value2, value: 10}
    edgePoints:
      - {type: role, text: admin}
      - {type: tombstone, value: 1}
`

type configImport struct {
	Nodes []NodeEdgeChildren
}

func TestConfigImport(t *testing.T) {
	var res configImport

	err := yaml.Unmarshal([]byte(configTestYAML), &res)
	if err != nil {
		t.Fatal("unmarshal failed: ", err)
	}

	if !reflect.DeepEqual(res.Nodes[0], configTestNode) {
		fmt.Printf("res: %+v\n", res)
		t.Fatal("Did not get expected result")
	}
}

/*
var configTestNodeChildren = NodeEdgeChildren{
	NodeEdge: NodeEdge{
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
*/
