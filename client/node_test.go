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

	y, err := client.ExportNodes(nc, root.Parent, root.ID)

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
var testReplaceNodes = data.NodeEdgeChildren{
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

func TestReplaceIDs(t *testing.T) {
	client.ReplaceIDs(&testReplaceNodes, "parent123")

	if testReplaceNodes.ID == "123" {
		t.Fatal("ID not replaced")
	}

	if testReplaceNodes.Children[0].ID == "abc" {
		t.Fatal("child ID not replaced")
	}

	// make sure nodes occur in multiple places, they have the same IDs
	if testReplaceNodes.ID != testReplaceNodes.Children[0].Children[0].ID {
		t.Fatal("123 IDs did not get replaced with the same value")
	}

	// mode sure any points of type nodeID get updated
	if testReplaceNodes.Children[0].Points[1].Text == "123" {
		t.Fatal("Points of type nodeID are not getting updated")
	}

	// make sure blank ids are handled correctly
	if testReplaceNodes.Children[0].ID == testReplaceNodes.Children[1].ID {
		t.Fatal("Blank node IDs not handled correctly")
	}

	// mode sure blank nodeID points are ignored
	if testReplaceNodes.Points[0].Text != "" {
		t.Fatal("Blank nodeID point not ignored")
	}

	if testReplaceNodes.Parent != "parent123" {
		t.Fatal("top level parent not correct")
	}

	if testReplaceNodes.ID != testReplaceNodes.Children[0].Parent {
		t.Fatal("child parent not correct")
	}

	fmt.Printf("CLIFF: testReplaceNodes: %+v\n", testReplaceNodes)
}
