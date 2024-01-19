package client_test

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// FIXME, need tests for duplicate, move, and mirror node

func TestExportNodes(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	y, err := client.ExportNodes(nc, root.ID)

	if err != nil {
		t.Fatal("Error exporting nodes: ", err)
	}

	// convert back to nodes and check a few
	var exp client.SiotExport

	err = yaml.Unmarshal(y, &exp)
	if err != nil {
		t.Fatal("Unmarshal error: ", err)
	}

	if len(exp.Nodes) < 1 {
		t.Fatal("no top level node")
	}

	if len(exp.Nodes[0].Children) < 1 {
		t.Fatal("no child nodes")
	}

	if exp.Nodes[0].Type != data.NodeTypeDevice {
		t.Fatal("top level node should be device")
	}

	if exp.Nodes[0].Children[0].Type != data.NodeTypeUser {
		t.Fatal("child node is not user type")
	}
}

var testImportNodesYaml = `
nodes:
- type: group
  points:
  - type: description
    text: "group 1"
  children:
  - type: variable
    points:
    - type: description
      text: var 1
    - type: value
      value: 10
`

func TestImportNodes(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	// make sure we can't import at a bogus place
	err = client.ImportNodes(nc, "bogusrootid", []byte(testImportNodesYaml), "test", false)
	if err == nil {
		t.Fatal("Should have gotten an error importing at bogus location")
	}

	err = client.ImportNodes(nc, root.ID, []byte(testImportNodesYaml), "test", false)
	if err != nil {
		t.Fatal("Error importing: ", err)
	}

	children, err := client.GetNodes(nc, root.ID, "all", "", false)
	if err != nil {
		t.Fatal("Error getting children: ", err)
	}

	if len(children) < 2 {
		t.Fatal("Should be at least 2 children")
	}

	var g data.NodeEdge

	for _, c := range children {
		if c.Type == data.NodeTypeGroup {
			g = c
			break
		}
	}

	if g.Type != data.NodeTypeGroup {
		t.Fatal("group node not found")
	}

	children, err = client.GetNodes(nc, g.ID, "all", "", false)
	if err != nil {
		t.Fatal("error getting group children")
	}

	if len(children) < 1 {
		t.Fatal("Group should have at least 1 child")
	}
}

var testImportNodesYamlWithIDs = `
nodes:
- type: group
  id: 111
  points:
  - type: description
    text: "group 1"
  children:
  - type: variable
    id: 222
    parent: 111
    points:
    - type: description
      text: var 1
    - type: value
      value: 10
`

func TestImportNodesPreserveIDs(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	err = client.ImportNodes(nc, root.ID, []byte(testImportNodesYamlWithIDs), "test", true)
	if err != nil {
		t.Fatal("Error importing: ", err)
	}

	children, err := client.GetNodes(nc, root.ID, "all", "", false)
	if err != nil {
		t.Fatal("Error getting children: ", err)
	}

	var g data.NodeEdge

	for _, c := range children {
		if c.Type == data.NodeTypeGroup {
			g = c
			break
		}
	}

	if g.Type != data.NodeTypeGroup {
		t.Fatal("group node not found")
	}

	if g.ID != "111" {
		t.Fatal("did not get expected group ID")
	}

	children, err = client.GetNodes(nc, g.ID, "all", "", false)
	if err != nil {
		t.Fatal("error getting group children")
	}

	if len(children) < 1 {
		t.Fatal("Group should have at least 1 child")
	}

}

var testImportNodesYamlBadParent = `
nodes:
- type: group
  id: 111
  points:
  - type: description
    text: "group 1"
  children:
  - type: variable
    id: 222
    parent: 123
    points:
    - type: description
      text: var 1
    - type: value
      value: 10
`

func TestImportNodesBadParent(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	err = client.ImportNodes(nc, root.ID, []byte(testImportNodesYamlBadParent), "test", true)
	if err == nil {
		t.Fatal("should have caught bad parent")
	}
}

func TestReplaceIDs(t *testing.T) {
	testNodes := data.NodeEdgeChildren{
		NodeEdge: data.NodeEdge{
			ID:   "123",
			Type: "testType",
			Points: []data.Point{
				{Type: "nodeID", Text: "", Key: "0"},
			},
		},
		Children: []data.NodeEdgeChildren{
			{NodeEdge: data.NodeEdge{
				ID:   "",
				Type: "testY",
				Points: []data.Point{
					{Type: "description", Text: "test Y1", Key: "2"},
					{Type: "nodeID", Text: "123", Key: "0"},
				},
				EdgePoints: []data.Point{
					{Type: "role", Text: "user"},
				},
			},
				Children: []data.NodeEdgeChildren{
					{NodeEdge: data.NodeEdge{
						ID:   "123",
						Type: "testY",
						Points: []data.Point{
							{Type: "description", Text: "test Y2"},
						},
					}, Children: nil},
				},
			},
			{NodeEdge: data.NodeEdge{
				ID:   "",
				Type: "testY",
				Points: []data.Point{
					{Type: "description", Text: "test Y1", Key: "2"},
					{Type: "nodeID", Text: "123", Key: "0"},
				},
				EdgePoints: []data.Point{
					{Type: "role", Text: "user"},
				},
			},
				Children: nil,
			},
		},
	}

	client.ReplaceIDs(&testNodes, "parent123")

	if testNodes.ID == "123" {
		t.Fatal("ID not replaced")
	}

	if testNodes.Children[0].ID == "abc" {
		t.Fatal("child ID not replaced")
	}

	// make sure nodes occur in multiple places, they have the same IDs
	if testNodes.ID != testNodes.Children[0].Children[0].ID {
		t.Fatal("123 IDs did not get replaced with the same value")
	}

	// mode sure any points of type nodeID get updated
	if testNodes.Children[0].Points[1].Text == "123" {
		t.Fatal("Points of type nodeID are not getting updated")
	}

	// make sure blank ids are handled correctly
	if testNodes.Children[0].ID == testNodes.Children[1].ID {
		t.Fatal("Blank node IDs not handled correctly")
	}

	// mode sure blank nodeID points are ignored
	if testNodes.Points[0].Text != "" {
		t.Fatal("Blank nodeID point not ignored")
	}

	if testNodes.Parent != "parent123" {
		t.Fatal("top level parent not correct")
	}

	if testNodes.ID != testNodes.Children[0].Parent {
		t.Fatal("child parent not correct")
	}
}
