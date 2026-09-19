package server

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// signIn signs a user in the way the UI does and returns the JWT.
func signIn(t *testing.T, nc *nats.Conn, email, pass string) string {
	t.Helper()
	for range 50 {
		nodes, err := client.UserCheck(nc, email, pass)
		if err != nil {
			t.Fatal("user check:", err)
		}
		for _, n := range nodes {
			if n.Type == data.NodeTypeJWT {
				if jwt, _ := n.Points.Text(data.PointTypeToken, ""); jwt != "" {
					return jwt
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("no JWT from sign-in")
	return ""
}

// TestUserEdgeScope covers the store's checks on what a signed-in user
// may write and read through the user namespace: both ends of an edge
// have to be in scope, an edge may not make a cycle, secrets are left out
// of replies, and a node's parents outside the user's groups are not
// listed.
func TestUserEdgeScope(t *testing.T) {
	opts := TestServerOptions
	opts.AuthToken = "tok"
	nc, root, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()

	for _, n := range []any{
		client.Group{ID: "G", Parent: root.ID, Description: "group g"},
		client.Group{ID: "H", Parent: root.ID, Description: "group h"},
		client.User{ID: "U", Parent: "G", FirstName: "u", Email: "u@example.com", Pass: "pw"},
		client.Variable{ID: "var-g", Parent: "G", Description: "in g"},
		client.Variable{ID: "var-h", Parent: "H", Description: "in h"},
		client.Sync{ID: "sync-g", Parent: "G", Description: "up", URI: "nats://x", AuthToken: "secret-token"},
		client.Variable{ID: "shared", Parent: "G", Description: "shared"},
	} {
		if err := client.SendNodeType(nc, n, "test"); err != nil {
			t.Fatal(err)
		}
	}
	// shared sits in both groups
	if err := client.MirrorNode(nc, "shared", "G", "H", "test"); err != nil {
		t.Fatal(err)
	}

	jwt := signIn(t, nc, "u@example.com", "pw")

	user, err := nats.Connect(fmt.Sprintf("nats://localhost:%v", opts.NatsPort),
		nats.UserInfo("U", jwt), nats.CustomInboxPrefix(client.InboxPrefix("U")),
		nats.NoReconnect())
	if err != nil {
		t.Fatal("user connect:", err)
	}
	defer user.Close()

	write := func(subj string, pts data.Points) string {
		t.Helper()
		m, err := user.Request(subj, pts.Encode(), 2*time.Second)
		if err != nil {
			t.Fatalf("%v: %v", subj, err)
		}
		return string(m.Data)
	}
	read := func(parent, id string, depth int) ([]data.NodeEdge, error) {
		var pts data.Points
		if depth > 0 {
			pts = append(pts, data.NewPointInt(data.PointTypeDepth, "", int64(depth)))
		}
		m, err := user.Request(fmt.Sprintf("u.G.U.nodes.%v.%v", parent, id), pts.Encode(), 2*time.Second)
		if err != nil {
			return nil, err
		}
		return data.DecodeNodes(m.Data)
	}

	live := data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeVariable),
	}

	// ---- grafting a node from another group under G is refused
	if got := write("u.G.U.ep.var-h.G", live); !strings.Contains(got, "not in scope") {
		t.Fatalf("graft of var-h got %q", got)
	}
	if _, err := read("all", "var-h", 0); err == nil || !strings.Contains(err.Error(), "not in scope") {
		t.Fatalf("var-h became readable: %v", err)
	}

	// ---- grafting the root under G is refused, and the store keeps
	// answering afterward
	rootPts := data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeDevice),
	}
	if got := write("u.G.U.ep."+root.ID+".G", rootPts); !strings.Contains(got, "not in scope") {
		t.Fatalf("graft of root got %q", got)
	}
	if _, err := read("G", "all", 100); err != nil {
		t.Fatal("deep read after refused graft:", err)
	}

	// ---- the cycle guard holds for a full-access writer as well
	err = client.SendEdgePoint(nc, root.ID, "G", data.NewPointFloat(data.PointTypeTombstone, "", 0), true)
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
	err = client.SendEdgePoint(nc, "G", "var-g", data.NewPointFloat(data.PointTypeTombstone, "", 0), true)
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}

	// ---- a new node under G is allowed, and is then in scope
	if got := write("u.G.U.ep.new-var.G", live); got != "" {
		t.Fatalf("new node refused: %q", got)
	}
	desc := data.Points{data.NewPointString(data.PointTypeDescription, "", "new")}
	if got := write("u.G.U.p.new-var.description.0", desc); got != "" {
		t.Fatalf("write to new node refused: %q", got)
	}

	// ---- a node the user deleted from G can be restored
	tomb := data.Points{data.NewPointFloat(data.PointTypeTombstone, "", 1)}
	if got := write("u.G.U.ep.var-g.G", tomb); got != "" {
		t.Fatalf("delete refused: %q", got)
	}
	if got := write("u.G.U.ep.var-g.G", live); got != "" {
		t.Fatalf("restore refused: %q", got)
	}

	// ---- secrets are left out of replies to the user
	nodes, err := read("G", "all", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range nodes {
		for _, p := range n.Points {
			if data.IsSecretPointType(p.Type) && p.Txt() != "" {
				t.Errorf("reply carried %v on %v: %q", p.Type, n.ID, p.Txt())
			}
		}
	}
	// the operator's own connection still reads them
	got, err := client.GetNodes(nc, "all", "sync-g", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if tok, _ := got[0].Points.Text(data.PointTypeAuthToken, ""); tok != "secret-token" {
		t.Fatalf("plain read lost the token: %q", tok)
	}

	// ---- a node's parents are listed only where they are in scope
	nodes, err = read("all", "shared", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || nodes[0].Parent != "G" {
		t.Fatalf("parents of shared: %v", nodes)
	}
}
