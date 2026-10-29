package server

import (
	"net/http"
	"testing"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// TestIDCheck: an ID chosen by hand that would break a subject or a stream
// name is refused by the API, the store, and at startup.
func TestIDCheck(t *testing.T) {
	opts := TestServerOptions
	nc, root, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	base := "http://localhost:" + opts.HTTPPort
	http.DefaultClient.CloseIdleConnections()
	token := apiLogin(t, base)

	node := data.NodeEdge{ID: "bad_id", Type: data.NodeTypeVariable, Parent: root.ID}
	code, body := apiDo(t, http.MethodPost, base+"/v1/nodes", token, node)
	if code != http.StatusBadRequest {
		t.Errorf("API insert with a bad ID: got %v, want 400: %s", code, body)
	}

	node.ID = "good-id"
	code, body = apiDo(t, http.MethodPost, base+"/v1/nodes", token, node)
	if code != http.StatusOK {
		t.Errorf("API insert with a good ID: got %v: %s", code, body)
	}

	// a client that skips SendNode and writes the edge directly
	err = client.SendEdgePoint(nc, "raw_id", root.ID,
		data.NewPointFloat(data.PointTypeTombstone, "", 0), true)
	if err == nil {
		t.Error("store accepted an edge with a bad ID")
	}
	nodes, err := client.GetNodes(nc, root.ID, "all", "", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range nodes {
		if n.ID == "raw_id" || n.ID == "bad_id" {
			t.Errorf("node %v was created", n.ID)
		}
	}

	// checked before anything starts, so no server is left running
	if _, _, err := NewServer(Options{ID: "bad_instance"}); err == nil {
		t.Error("server accepted a bad instance ID")
	}
}
