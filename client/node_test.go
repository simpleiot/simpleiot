package client_test

import (
	"fmt"
	"testing"

	"github.com/simpleiot/simpleiot/client"
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

	fmt.Println("CLIFF: yaml: ", string(y))
}
