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
