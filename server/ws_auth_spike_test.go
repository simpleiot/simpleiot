package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// This file is the Phase 0 spike for the UI over NATS, kept as the
// regression test for browser connections: a nats.go client arriving
// through the HTTP server's WebSocket proxy with the user ID and JWT as
// user and password is accepted by the authorizer, scoped to the user's
// anchors by the server, and scoped to the anchor's subtree by the store.

// TestUserDeadline pins down that a connection registered with a deadline
// is closed by the NATS server when it passes, which is how a JWT's expiry
// is enforced on a live connection.
func TestUserDeadline(t *testing.T) {
	a, _ := spikeAuthorizer(t, "tok")
	a.users = fakeUsers{
		id:      "U",
		expires: time.Now().Add(2 * time.Second),
		anchors: map[string][]string{"U": {"G"}},
	}
	_, _, wsURL := spikeServer(t, a)

	nc, err := nats.Connect(wsURL, nats.UserInfo("U", fakeJWT), nats.NoReconnect())
	if err != nil {
		t.Fatal("user connect:", err)
	}
	defer nc.Close()

	deadline := time.Now().Add(6 * time.Second)
	for nc.IsConnected() && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	if nc.IsConnected() {
		t.Fatal("connection still open after the token expired")
	}
}

// connzUsers lists the authorized user of every connection the server
// reports on its monitoring port.
func connzUsers(t *testing.T) []string {
	t.Helper()
	resp, err := http.Get(fmt.Sprintf("http://localhost:%v/connz?auth=1",
		TestServerOptions.NatsHTTPPort))
	if err != nil {
		t.Fatal("connz:", err)
	}
	defer resp.Body.Close()
	var cz struct {
		Conns []struct {
			AuthorizedUser string `json:"authorized_user"`
		} `json:"connections"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cz); err != nil {
		t.Fatal("decode connz:", err)
	}
	var users []string
	for _, c := range cz.Conns {
		users = append(users, c.AuthorizedUser)
	}
	return users
}

func waitDisconnected(t *testing.T, nc *nats.Conn, what string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for nc.IsConnected() && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	if nc.IsConnected() {
		t.Fatalf("%v: connection still open", what)
	}
}

func TestUserOverWebSocket(t *testing.T) {
	opts := TestServerOptions
	opts.AuthToken = "tok"
	nc, root, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()

	wsURL := fmt.Sprintf("ws://localhost:%v/", opts.HTTPPort)

	// group G holds the user; group H is out of scope. G has a chain
	// three deep for the depth test.
	for _, n := range []any{
		client.Group{ID: "G", Parent: root.ID, Description: "group g"},
		client.Group{ID: "H", Parent: root.ID, Description: "group h"},
		client.User{ID: "U", Parent: "G", FirstName: "u", Email: "u@example.com", Pass: "pw"},
		client.Variable{ID: "var-g", Parent: "G", Description: "in g"},
		client.Variable{ID: "var-h", Parent: "H", Description: "in h"},
		client.Group{ID: "G1", Parent: "G", Description: "child"},
		client.Group{ID: "G2", Parent: "G1", Description: "grandchild"},
		client.Group{ID: "G3", Parent: "G2", Description: "great-grandchild"},
	} {
		if err := client.SendNodeType(nc, n, "test"); err != nil {
			t.Fatal(err)
		}
	}

	// ---- sign in the way the UI does, and connect with the JWT
	var jwt string
	for range 50 {
		nodes, err := client.UserCheck(nc, "u@example.com", "pw")
		if err != nil {
			t.Fatal("user check:", err)
		}
		for _, n := range nodes {
			if n.Type == data.NodeTypeJWT {
				jwt, _ = n.Points.Text(data.PointTypeToken, "")
			}
		}
		if jwt != "" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if jwt == "" {
		t.Fatal("no JWT from sign-in")
	}

	var errs errCollector
	user, err := nats.Connect(wsURL, nats.UserInfo("U", jwt),
		nats.CustomInboxPrefix(userInboxPrefix("U")),
		nats.ErrorHandler(errs.handler), nats.NoReconnect())
	if err != nil {
		t.Fatal("user connect through proxy:", err)
	}
	defer user.Close()

	// ---- what is refused: a bad or foreign JWT, no credentials at all
	for _, tt := range []struct {
		name string
		opts []nats.Option
	}{
		{"invalid jwt", []nats.Option{nats.UserInfo("U", "eyJhbGciOiJIUzI1NiJ9.bad.sig")}},
		{"jwt for another user", []nats.Option{nats.UserInfo("V", jwt)}},
		{"anonymous", nil},
		{"wrong token", []nats.Option{nats.Token("bad")}},
	} {
		if _, err := nats.Connect(wsURL, append(tt.opts, nats.NoReconnect())...); !errors.Is(err, nats.ErrAuthorization) {
			t.Fatalf("%v: expected refusal, got %v", tt.name, err)
		}
	}

	// the shared token still works on the WebSocket listener
	tokenConn, err := nats.Connect(wsURL, nats.Token("tok"), nats.NoReconnect())
	if err != nil {
		t.Fatal("token over websocket:", err)
	}
	tokenConn.Close()

	// ---- the connection is reported under the user ID
	found := false
	for _, u := range connzUsers(t) {
		if u == "U" {
			found = true
		}
	}
	if !found {
		t.Fatal("user connection not reported by Connz")
	}

	// ---- auth.me answers with the user's anchors
	me, err := user.Request("auth.me", []byte(jwt), time.Second)
	if err != nil {
		t.Fatal("auth.me:", err)
	}
	meNodes, err := data.DecodeNodes(me.Data)
	if err != nil {
		t.Fatal("auth.me decode:", err)
	}
	if len(meNodes) != 1 || meNodes[0].ID != "U" || meNodes[0].Parent != "G" {
		t.Fatalf("auth.me returned %v", meNodes)
	}
	if _, ok := meNodes[0].Points.Find(data.PointTypePass, ""); ok {
		t.Fatal("auth.me returned the password hash")
	}
	me, err = user.Request("auth.me", []byte("eyJ.bad.sig"), time.Second)
	if err != nil {
		t.Fatal("auth.me:", err)
	}
	if _, err := data.DecodeNodes(me.Data); err == nil {
		t.Fatal("auth.me accepted an invalid token")
	}

	// ---- reads: under G through the user namespace, refused elsewhere
	getNodes := func(anchor, parent, id string, depth int) ([]data.NodeEdge, error) {
		var pts data.Points
		if depth > 0 {
			pts = append(pts, data.NewPointInt(data.PointTypeDepth, "", int64(depth)))
		}
		subj := fmt.Sprintf("u.%v.U.nodes.%v.%v", anchor, parent, id)
		m, err := user.Request(subj, pts.Encode(), time.Second)
		if err != nil {
			return nil, err
		}
		return data.DecodeNodes(m.Data)
	}

	nodes, err := getNodes("G", "G", "all", 0)
	if err != nil {
		t.Fatal("children of G:", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 children of G, got %v", len(nodes))
	}

	nodes, err = getNodes("G", "all", "G", 0)
	if err != nil || len(nodes) != 1 || nodes[0].ID != "G" {
		t.Fatalf("anchor fetch: %v %v", nodes, err)
	}

	if _, err := getNodes("G", "H", "all", 0); err == nil || !strings.Contains(err.Error(), "not in scope") {
		t.Fatalf("expected not in scope for H, got %v", err)
	}
	if _, err := getNodes("G", "all", "var-h", 0); err == nil || !strings.Contains(err.Error(), "not in scope") {
		t.Fatalf("expected not in scope for var-h, got %v", err)
	}
	// the plain subject and a foreign anchor are outside the permission set
	if _, err := user.Request("nodes.H.all", nil, 300*time.Millisecond); err == nil {
		t.Fatal("plain nodes request was answered")
	}
	if _, err := getNodes("H", "H", "all", 0); err == nil {
		t.Fatal("request under a foreign anchor was answered")
	}

	// ---- depth: children and grandchildren, nothing deeper
	nodes, err = getNodes("G", "G", "all", 1)
	if err != nil {
		t.Fatal("depth 1:", err)
	}
	ids := map[string]bool{}
	for _, n := range nodes {
		ids[n.ID] = true
	}
	for _, want := range []string{"U", "var-g", "G1", "G2"} {
		if !ids[want] {
			t.Errorf("depth 1 reply is missing %v: %v", want, ids)
		}
	}
	if ids["G3"] {
		t.Error("depth 1 reply went three levels deep")
	}

	// ---- live points under G arrive; a subscription under H is refused
	sub, err := user.SubscribeSync("up.G.>")
	if err != nil {
		t.Fatal(err)
	}
	if err := user.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := client.SendNodePoint(nc, "var-g", data.NewPointFloat(data.PointTypeValue, "", 42), true); err != nil {
		t.Fatal(err)
	}
	m, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatal("no point under G:", err)
	}
	if m.Subject != "up.G.var-g.value.0" {
		t.Fatalf("unexpected subject %v", m.Subject)
	}
	if _, err := user.SubscribeSync("up.H.>"); err == nil {
		_ = user.Flush()
	}
	if _, err := user.SubscribeSync("_INBOX.>"); err == nil {
		_ = user.Flush()
	}
	time.Sleep(200 * time.Millisecond)
	if !errs.saw("Permissions Violation") {
		t.Fatalf("expected a permissions violation, got %v", errs.errs)
	}

	// ---- writes: origin is the user whatever the point says
	pt := data.NewPointFloat(data.PointTypeValue, "", 7)
	pt.Origin = "someone-else"
	pts := data.Points{pt}
	m, err = user.Request("u.G.U.p.var-g.value.0", pts.Encode(), time.Second)
	if err != nil {
		t.Fatal("write under G:", err)
	}
	if len(m.Data) != 0 {
		t.Fatalf("write under G refused: %s", m.Data)
	}
	got, err := client.GetNodes(nc, "all", "var-g", "", false)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := got[0].Points.Find(data.PointTypeValue, "")
	if p.Val() != 7 || p.Origin != "U" {
		t.Fatalf("stored point %v, want value 7 from origin U", p)
	}

	// a write outside G is refused by the store and does not land
	m, err = user.Request("u.G.U.p.var-h.value.0", pts.Encode(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(m.Data), "not in scope") {
		t.Fatalf("write under H got %q", m.Data)
	}
	got, err = client.GetNodes(nc, "all", "var-h", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got[0].Points.Find(data.PointTypeValue, ""); ok {
		t.Fatal("point landed on a node outside the user's scope")
	}

	// an edge point may only be written under an in-scope parent
	tomb := data.Points{data.NewPointFloat(data.PointTypeTombstone, "", 1)}
	m, err = user.Request("u.G.U.ep.var-h.H", tomb.Encode(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(m.Data), "not in scope") {
		t.Fatalf("edge write under H got %q", m.Data)
	}

	// ---- revocation: removing the user from the group closes the
	// connection, and it cannot come back until the user is restored
	if err := client.DeleteNode(nc, "U", "G", "test"); err != nil {
		t.Fatal("delete user:", err)
	}
	waitDisconnected(t, user, "user removed from group")

	if _, err := nats.Connect(wsURL, nats.UserInfo("U", jwt), nats.NoReconnect()); !errors.Is(err, nats.ErrAuthorization) {
		t.Fatalf("expected removed user to be refused, got %v", err)
	}

	if err := client.SendEdgePoint(nc, "U", "G",
		data.NewPointFloat(data.PointTypeTombstone, "", 0), true); err != nil {
		t.Fatal("restore user:", err)
	}
	again, err := nats.Connect(wsURL, nats.UserInfo("U", jwt),
		nats.CustomInboxPrefix(userInboxPrefix("U")), nats.NoReconnect())
	if err != nil {
		t.Fatal("reconnect after restore:", err)
	}
	defer again.Close()

	// ---- a new anchor closes the connection too, so it reconnects with
	// both groups; a password change closes it for good
	if err := client.MirrorNode(nc, "U", "G", "H", "test"); err != nil {
		t.Fatal("mirror user:", err)
	}
	waitDisconnected(t, again, "anchor added")

	both, err := nats.Connect(wsURL, nats.UserInfo("U", jwt),
		nats.CustomInboxPrefix(userInboxPrefix("U")), nats.NoReconnect())
	if err != nil {
		t.Fatal("reconnect with two anchors:", err)
	}
	defer both.Close()
	if _, err := both.Request("u.H.U.nodes.H.all", nil, time.Second); err != nil {
		t.Fatal("request under the new anchor:", err)
	}

	if err := client.SendNodePoint(nc, "U", data.NewPointString(data.PointTypePass, "", "new"), true); err != nil {
		t.Fatal(err)
	}
	waitDisconnected(t, both, "password changed")
}

// TestOpenInstanceWebSocket: with no token configured, an anonymous
// WebSocket connection has full access, while a user JWT is still scoped.
func TestOpenInstanceWebSocket(t *testing.T) {
	nc, root, stop, err := TestServer()
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()

	wsURL := fmt.Sprintf("ws://localhost:%v/", TestServerOptions.HTTPPort)

	anon, err := nats.Connect(wsURL, nats.NoReconnect())
	if err != nil {
		t.Fatal("anonymous connect:", err)
	}
	defer anon.Close()
	if _, err := client.GetNodes(anon, "root", "all", "", false); err != nil {
		t.Fatal("anonymous request:", err)
	}

	var jwt string
	for range 50 {
		nodes, err := client.UserCheck(nc, "admin", "admin")
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range nodes {
			if n.Type == data.NodeTypeJWT {
				jwt, _ = n.Points.Text(data.PointTypeToken, "")
			}
		}
		if jwt != "" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	users, err := client.GetNodes(nc, "all", "all", data.NodeTypeUser, false)
	if err != nil || len(users) == 0 {
		t.Fatal("admin user:", err)
	}
	adminID := users[0].ID

	var errs errCollector
	user, err := nats.Connect(wsURL, nats.UserInfo(adminID, jwt),
		nats.CustomInboxPrefix(userInboxPrefix(adminID)),
		nats.ErrorHandler(errs.handler), nats.NoReconnect())
	if err != nil {
		t.Fatal("user connect:", err)
	}
	defer user.Close()

	// admin sits under the root node, so that is the anchor
	subj := fmt.Sprintf("u.%v.%v.nodes.%v.all", root.ID, adminID, root.ID)
	if _, err := user.Request(subj, nil, time.Second); err != nil {
		t.Fatal("request under root anchor:", err)
	}
	if _, err := user.SubscribeSync("p.>"); err == nil {
		_ = user.Flush()
	}
	time.Sleep(200 * time.Millisecond)
	if !errs.saw("Permissions Violation") {
		t.Fatal("user JWT not scoped on an open instance")
	}
}
