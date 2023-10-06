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

var configTestNodeYAML = `
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

	err := yaml.Unmarshal([]byte(configTestNodeYAML), &res)
	if err != nil {
		t.Fatal("unmarshal failed: ", err)
	}

	if !reflect.DeepEqual(res.Nodes[0], configTestNode) {
		fmt.Printf("res: %+v\n", res)
		t.Fatal("Did not get expected result")
	}
}

var configTestNodeChildren = NodeEdgeChildren{
	NodeEdge{
		ID:     "123",
		Parent: "456",
		Type:   "testType",
	},
	[]NodeEdgeChildren{
		{NodeEdge{
			ID:     "abc",
			Parent: "123",
			Type:   "testY",
			Points: []Point{
				{Type: "description", Text: "test Y1", Key: "2"},
			},
			EdgePoints: []Point{
				{Type: "role", Text: "user"},
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
				}, nil},
			},
		},
	},
}

var configTestNodeChildrenYAML = `
nodes:
  - type: testType
    id: 123
    parent: 456
    children:
      - id: "abc"
        parent: "123"
        type: "testY"
        points:
          - {type: description, text: "test Y1", key: "2"}
        edgePoints:
          - {type: role, text: user}
        children:
          - id: "jkl"
            parent: "abc"
            type: "testY"
            points:
              - {type: description, text: "test Y2"}
`

func TestConfigChildrenImport(t *testing.T) {
	var res configImport

	err := yaml.Unmarshal([]byte(configTestNodeChildrenYAML), &res)
	if err != nil {
		t.Fatal("unmarshal failed: ", err)
	}

	if !reflect.DeepEqual(res.Nodes[0], configTestNodeChildren) {
		fmt.Printf("res: %+v\n", res)
		t.Fatal("Did not get expected result")
	}
}
