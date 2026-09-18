package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// apiLoginAs signs a user in over HTTP and returns the JWT.
func apiLoginAs(t *testing.T, base, email, pass string) string {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.PostForm(base+"/v1/auth",
			url.Values{"email": {email}, "password": {pass}})
		if err != nil {
			t.Fatal("login:", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var auth data.Auth
		if resp.StatusCode == http.StatusOK &&
			json.Unmarshal(body, &auth) == nil && auth.Token != "" {
			return auth.Token
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("login did not succeed")
	return ""
}

// apiRaw sends a request with the Authorization header exactly as given.
func apiRaw(t *testing.T, method, url, authHeader string, body []byte) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

// TestAPINoTokenNoBypass: an instance with no auth token configured does
// not treat a request with no credentials as the shared token.
func TestAPINoTokenNoBypass(t *testing.T) {
	opts := TestServerOptions
	opts.AuthToken = ""
	_, root, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()
	http.DefaultClient.CloseIdleConnections()

	base := "http://localhost:" + opts.HTTPPort
	pts, _ := json.Marshal(data.Points{data.NewPointFloat(data.PointTypeValue, "", 1)})

	for _, hdr := range []string{"", "Bearer ", "Bearer x"} {
		code, _ := apiRaw(t, http.MethodPost, base+"/v1/nodes/"+root.ID+"/points", hdr, pts)
		if code != http.StatusUnauthorized {
			t.Errorf("POST points with header %q: got %v, want 401", hdr, code)
		}
		code, _ = apiRaw(t, http.MethodGet, base+"/v1/nodes/"+root.ID, hdr, []byte("all"))
		if code != http.StatusUnauthorized {
			t.Errorf("GET with header %q: got %v, want 401", hdr, code)
		}
	}

	// a signed-in user still works on an open instance
	jwt := apiLogin(t, base)
	code, _ := apiGetNode(t, base+"/v1/nodes/"+root.ID, jwt)
	if code != http.StatusOK {
		t.Errorf("admin GET root: %v", code)
	}
}

// TestAPIUserScope: the node routes keep a user inside the user's groups,
// refuse a deleted user's token, leave secrets out of replies, and carry
// the response headers.
func TestAPIUserScope(t *testing.T) {
	opts := TestServerOptions
	opts.AuthToken = "tok"
	nc, root, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()
	http.DefaultClient.CloseIdleConnections()

	base := "http://localhost:" + opts.HTTPPort

	for _, n := range []any{
		client.Group{ID: "G", Parent: root.ID, Description: "group g"},
		client.Group{ID: "H", Parent: root.ID, Description: "group h"},
		client.User{ID: "U", Parent: "G", FirstName: "u", Email: "u@example.com", Pass: "pw"},
		client.Variable{ID: "var-g", Parent: "G", Description: "in g"},
		client.Variable{ID: "var-g2", Parent: "G", Description: "in g too"},
		client.Group{ID: "G1", Parent: "G", Description: "sub g"},
		client.Variable{ID: "var-h", Parent: "H", Description: "in h"},
		client.Sync{ID: "sync-g", Parent: "G", Description: "up", URI: "nats://x", AuthToken: "secret-token"},
		client.EnrollToken{ID: "et-h", Parent: "H", Description: "token in h"},
	} {
		if err := client.SendNodeType(nc, n, "test"); err != nil {
			t.Fatal(err)
		}
	}

	jwt := apiLoginAs(t, base, "u@example.com", "pw")
	pts := data.Points{data.NewPointFloat(data.PointTypeValue, "", 1)}
	not := data.Notification{Subject: "hi", Message: "there"}

	type op struct {
		name   string
		method string
		path   string
		body   any
	}
	// every route, aimed at H
	refused := []op{
		{"get", http.MethodGet, "/v1/nodes/var-h", nil},
		{"points", http.MethodPost, "/v1/nodes/var-h/points", pts},
		{"delete", http.MethodDelete, "/v1/nodes/var-h", api.NodeDelete{Parent: "H"}},
		{"move out", http.MethodPost, "/v1/nodes/var-g/parents", api.NodeMove{OldParent: "G", NewParent: "H"}},
		{"move in", http.MethodPost, "/v1/nodes/var-h/parents", api.NodeMove{OldParent: "H", NewParent: "G"}},
		{"mirror in", http.MethodPut, "/v1/nodes/var-h/parents", api.NodeCopy{OldParent: "H", NewParent: "G"}},
		{"duplicate in", http.MethodPut, "/v1/nodes/var-h/parents", api.NodeCopy{NewParent: "G", Duplicate: true}},
		{"notify", http.MethodPost, "/v1/nodes/var-h/not", not},
		{"key", http.MethodPost, "/v1/nodes/et-h/key", nil},
		{"insert under H", http.MethodPost, "/v1/nodes", data.NodeEdge{Parent: "H", Type: data.NodeTypeVariable}},
		// attaching an existing node from H under G by naming its ID
		{"graft", http.MethodPost, "/v1/nodes", data.NodeEdge{ID: "var-h", Parent: "G", Type: data.NodeTypeVariable}},
		{"graft root", http.MethodPost, "/v1/nodes", data.NodeEdge{ID: root.ID, Parent: "G", Type: data.NodeTypeDevice}},
	}
	for _, o := range refused {
		code, body := apiDo(t, o.method, base+o.path, jwt, o.body)
		if code != http.StatusForbidden {
			t.Errorf("%v: got %v %s, want 403", o.name, code, body)
		}
	}
	// nothing landed
	if got, _ := client.GetNodes(nc, "all", "var-h", "", false); len(got) != 1 || got[0].Parent != "H" {
		t.Errorf("var-h edges changed: %v", got)
	}
	if got, _ := client.GetNodes(nc, "G", "var-h", "", false); len(got) != 0 {
		t.Error("var-h was attached under G")
	}

	// the same routes inside G
	if code, body := apiGetNode(t, base+"/v1/nodes/var-g", jwt); code != http.StatusOK {
		t.Errorf("get var-g: got %v %s, want 200", code, body)
	}
	allowed := []op{
		{"points", http.MethodPost, "/v1/nodes/var-g/points", pts},
		{"move", http.MethodPost, "/v1/nodes/var-g/parents", api.NodeMove{OldParent: "G", NewParent: "G1"}},
		{"mirror", http.MethodPut, "/v1/nodes/var-g2/parents", api.NodeCopy{OldParent: "G", NewParent: "G1"}},
		{"duplicate", http.MethodPut, "/v1/nodes/var-g2/parents", api.NodeCopy{NewParent: "G1", Duplicate: true}},
		{"notify", http.MethodPost, "/v1/nodes/var-g/not", not},
		{"insert", http.MethodPost, "/v1/nodes", data.NodeEdge{Parent: "G", Type: data.NodeTypeVariable}},
		{"delete", http.MethodDelete, "/v1/nodes/var-g", api.NodeDelete{Parent: "G1"}},
	}
	for _, o := range allowed {
		code, body := apiDo(t, o.method, base+o.path, jwt, o.body)
		if code != http.StatusOK {
			t.Errorf("%v: got %v %s, want 200", o.name, code, body)
		}
	}

	// secrets are left out of a user's reads, parents outside G are not
	// listed, and the anchor itself is readable
	code, body := apiGetNode(t, base+"/v1/nodes/sync-g", jwt)
	if code != http.StatusOK {
		t.Fatalf("get sync-g: %v", code)
	}
	var nodes []data.NodeEdge
	if err := json.Unmarshal(body, &nodes); err != nil || len(nodes) != 1 {
		t.Fatalf("sync-g reply: %s", body)
	}
	if tok, _ := nodes[0].Points.Text(data.PointTypeAuthToken, ""); tok != "" {
		t.Errorf("reply carried the auth token: %q", tok)
	}
	code, body = apiGetNode(t, base+"/v1/nodes/G", jwt)
	if code != http.StatusOK {
		t.Fatalf("get anchor: %v %s", code, body)
	}
	if err := json.Unmarshal(body, &nodes); err != nil || len(nodes) != 1 || nodes[0].ID != "G" {
		t.Fatalf("anchor reply: %s", body)
	}
	// the shared token still reads everything
	code, body = apiGetNode(t, base+"/v1/nodes/sync-g", "")
	if code != http.StatusUnauthorized {
		t.Fatalf("empty bearer accepted: %v", code)
	}
	code, body = apiRaw(t, http.MethodGet, base+"/v1/nodes/sync-g", "tok", []byte("all"))
	if err := json.Unmarshal(body, &nodes); code != http.StatusOK || err != nil || len(nodes) != 1 {
		t.Fatalf("token read: %v %s", code, body)
	}
	if tok, _ := nodes[0].Points.Text(data.PointTypeAuthToken, ""); tok != "secret-token" {
		t.Errorf("token read lost the secret: %q", tok)
	}

	// response headers and body limit
	req, _ := http.NewRequest(http.MethodGet, base+"/", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	for _, h := range []string{"Content-Security-Policy", "X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy"} {
		if resp.Header.Get(h) == "" {
			t.Errorf("missing header %v", h)
		}
	}
	// valid JSON up to the limit, so the size is what stops it
	big := append([]byte(`[{"type":"description","text":"`), bytes.Repeat([]byte("x"), 5<<20)...)
	big = append(big, []byte(`"}]`)...)
	code, _ = apiRaw(t, http.MethodPost, base+"/v1/nodes/var-g2/points", "Bearer "+jwt, big)
	if code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized body: got %v, want 413", code)
	}

	// a deleted user's token is refused at once
	if err := client.DeleteNode(nc, "U", "G", "test"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		code, _ = apiGetNode(t, base+"/v1/nodes/var-g2", jwt)
		if code == http.StatusUnauthorized || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if code != http.StatusUnauthorized {
		t.Errorf("deleted user: got %v, want 401", code)
	}
}

// lanAddr returns a non-loopback IPv4 address of this machine, or "".
func lanAddr() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok && !ipn.IP.IsLoopback() && ipn.IP.To4() != nil {
			return ipn.IP.String()
		}
	}
	return ""
}

// TestAPIRequiredThroughProxy: under device auth required, the shared
// token is refused from a remote address through the HTTP port's
// WebSocket proxy, which would otherwise make the connection look local.
func TestAPIRequiredThroughProxy(t *testing.T) {
	lan := lanAddr()
	if lan == "" {
		t.Skip("no non-loopback address on this machine")
	}

	opts := TestServerOptions
	opts.AuthToken = "tok"
	opts.DeviceAuth = DeviceAuthRequired
	_, _, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()

	// from loopback, both the NATS port and the proxy accept the token
	for _, u := range []string{
		fmt.Sprintf("nats://localhost:%v", opts.NatsPort),
		fmt.Sprintf("ws://localhost:%v/", opts.HTTPPort),
	} {
		c, err := nats.Connect(u, nats.Token("tok"), nats.NoReconnect())
		if err != nil {
			t.Fatalf("%v: %v", u, err)
		}
		c.Close()
	}

	// from the LAN address, neither does
	for _, u := range []string{
		fmt.Sprintf("nats://%v:%v", lan, opts.NatsPort),
		fmt.Sprintf("ws://%v:%v/", lan, opts.HTTPPort),
	} {
		c, err := nats.Connect(u, nats.Token("tok"), nats.NoReconnect(), nats.Timeout(3*time.Second))
		if err == nil {
			// the proxy may close the socket after the handshake;
			// a connection that cannot make a request is refused too
			_, rerr := c.Request("nodes.root.all", nil, time.Second)
			c.Close()
			if rerr == nil {
				t.Fatalf("%v: token accepted from %v", u, lan)
			}
			err = rerr
		}
		if !strings.Contains(strings.ToLower(err.Error()), "auth") &&
			!strings.Contains(strings.ToLower(err.Error()), "closed") &&
			!strings.Contains(strings.ToLower(err.Error()), "eof") &&
			!strings.Contains(strings.ToLower(err.Error()), "connection") {
			t.Logf("%v refused with: %v", u, err)
		}
	}

	// a signed-in user through the proxy from the LAN still works
	base := "http://localhost:" + opts.HTTPPort
	http.DefaultClient.CloseIdleConnections()
	jwt := apiLogin(t, base)
	nodes, err := client.UserCheck(nil, "", "")
	_ = nodes
	_ = err
	admin := ""
	{
		// the admin user's ID is on the token's claims; read it back
		// through auth.me over the proxy
		c, err := nats.Connect(fmt.Sprintf("ws://localhost:%v/", opts.HTTPPort), nats.Token("tok"), nats.NoReconnect())
		if err != nil {
			t.Fatal(err)
		}
		m, err := c.Request("auth.me", []byte(jwt), time.Second)
		c.Close()
		if err != nil {
			t.Fatal(err)
		}
		me, err := data.DecodeNodes(m.Data)
		if err != nil || len(me) == 0 {
			t.Fatalf("auth.me: %v %v", me, err)
		}
		admin = me[0].ID
	}
	c, err := nats.Connect(fmt.Sprintf("ws://%v:%v/", lan, opts.HTTPPort),
		nats.UserInfo(admin, jwt), nats.CustomInboxPrefix(client.InboxPrefix(admin)),
		nats.NoReconnect())
	if err != nil {
		t.Fatalf("user through proxy from %v: %v", lan, err)
	}
	c.Close()
}
