package server

import (
	"crypto/tls"
	"encoding/base64"
	"net"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/simpleiot/simpleiot/client"
)

func TestDevicePermissions(t *testing.T) {
	p := devicePermissions("UKEY", "X", "R", []string{"R", "R2"})

	wantPub := []string{
		"nodes.root.all",
		"nodes.all.X",
		"ep.X.R",
		"inst.X.X.>",
		"$JS.API.STREAM.INFO.inst_X_X",
		"$JS.API.STREAM.CREATE.inst_X_X",
		"$JS.API.STREAM.NAMES",
		"$JS.API.STREAM.INFO.inst_X_R",
		"$JS.API.CONSUMER.CREATE.inst_X_R.>",
		"$JS.API.CONSUMER.INFO.inst_X_R.*",
		"$JS.API.CONSUMER.MSG.NEXT.inst_X_R.*",
		"$JS.ACK.inst_X_R.>",
		"$JS.API.STREAM.INFO.inst_X_R2",
		"$JS.API.CONSUMER.CREATE.inst_X_R2.>",
		"$JS.API.CONSUMER.INFO.inst_X_R2.*",
		"$JS.API.CONSUMER.MSG.NEXT.inst_X_R2.*",
		"$JS.ACK.inst_X_R2.>",
	}

	if !slices.Equal(p.Publish.Allow, wantPub) {
		t.Errorf("publish allow:\n got %v\nwant %v", p.Publish.Allow, wantPub)
	}

	// the device's own inbox, not the _INBOX.> every client shares
	if !slices.Equal(p.Subscribe.Allow, []string{"_INBOX_UKEY.>"}) {
		t.Errorf("subscribe allow: got %v", p.Subscribe.Allow)
	}

	if p.Publish.Deny != nil || p.Subscribe.Deny != nil {
		t.Error("expected allow lists only")
	}

	// nothing a device must never touch is in the list
	for _, s := range p.Publish.Allow {
		for _, bad := range []string{"p.", "up.", "auth.", "admin.", "inst.Y", "$JS.API.STREAM.LIST"} {
			if len(s) >= len(bad) && s[:len(bad)] == bad {
				t.Errorf("publish allow includes %v", s)
			}
		}
	}

	// the device's own ID is never an origin to pull from
	p = devicePermissions("UKEY", "X", "R", []string{"X", "R"})
	if slices.Contains(p.Publish.Allow, "$JS.API.STREAM.INFO.inst_X_X.>") {
		t.Error("device's own stream listed as a pull origin")
	}
}

// fakeClient is a ClientAuthentication for exercising Check without a
// server.
type fakeClient struct {
	opts  server.ClientOpts
	addr  net.Addr
	nonce []byte
	user  *server.User
}

func (f *fakeClient) GetOpts() *server.ClientOpts                 { return &f.opts }
func (f *fakeClient) GetTLSConnectionState() *tls.ConnectionState { return nil }
func (f *fakeClient) RegisterUser(u *server.User)                 { f.user = u }
func (f *fakeClient) RemoteAddress() net.Addr                     { return f.addr }
func (f *fakeClient) GetNonce() []byte                            { return f.nonce }
func (f *fakeClient) Kind() int                                   { return server.CLIENT }
func (f *fakeClient) GetID() uint64                               { return 1 }

func tcpAddr(s string) net.Addr {
	a, err := net.ResolveTCPAddr("tcp", s)
	if err != nil {
		panic(err)
	}
	return a
}

func TestCheckToken(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		deviceAuth string
		present    string
		addr       string
		want       bool
	}{
		{"open, nothing", "", "", "", "10.1.1.1:1", true},
		{"open, anything", "", "", "x", "10.1.1.1:1", true},
		{"token, right", "tok", "", "tok", "10.1.1.1:1", true},
		{"token, wrong", "tok", "", "bad", "10.1.1.1:1", false},
		{"token, missing", "tok", "", "", "10.1.1.1:1", false},
		{"required, remote", "tok", DeviceAuthRequired, "tok", "10.1.1.1:1", false},
		{"required, loopback", "tok", DeviceAuthRequired, "tok", "127.0.0.1:1", true},
		{"required, loopback v6", "tok", DeviceAuthRequired, "tok", "[::1]:1", true},
		{"required, wrong token on loopback", "tok", DeviceAuthRequired, "bad", "127.0.0.1:1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newAuthorizer(tt.token, tt.deviceAuth)
			c := &fakeClient{addr: tcpAddr(tt.addr)}
			c.opts.Token = tt.present
			if got := a.Check(c); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if c.user != nil {
				t.Error("token connections keep full access, no user should be registered")
			}
		})
	}
}

func TestCheckNkey(t *testing.T) {
	kp, _ := nkeys.CreateUser()
	pub, _ := kp.PublicKey()
	nonce := []byte("nonce-1234567890")
	sig, _ := kp.Sign(nonce)

	newClient := func() *fakeClient {
		c := &fakeClient{addr: tcpAddr("10.1.1.1:1"), nonce: nonce}
		c.opts.Nkey = pub
		c.opts.Sig = string(encodeSig(sig))
		return c
	}

	a := newAuthorizer("tok", "")
	a.rootID = "R"

	if a.Check(newClient()) {
		t.Fatal("accepted a key before the index was loaded")
	}

	a.ready = true
	if a.Check(newClient()) {
		t.Fatal("accepted an unknown key")
	}

	a.creds[pub] = &credEntry{credID: "c", deviceID: "X"}
	c := newClient()
	if !a.Check(c) {
		t.Fatal("refused a known key")
	}
	if c.user == nil || c.user.Permissions == nil ||
		!slices.Contains(c.user.Permissions.Publish.Allow, "inst.X.X.>") {
		t.Fatalf("device permissions not registered: %+v", c.user)
	}

	a.creds[pub].disabled = true
	if a.Check(newClient()) {
		t.Fatal("accepted a disabled key")
	}
	a.creds[pub].disabled = false

	// a signature over something other than the nonce is refused
	c = newClient()
	other, _ := kp.Sign([]byte("something else"))
	c.opts.Sig = string(encodeSig(other))
	if a.Check(c) {
		t.Fatal("accepted a bad signature")
	}

	// a key with no signature is refused even when known
	c = newClient()
	c.opts.Sig = ""
	if a.Check(c) {
		t.Fatal("accepted a key with no signature")
	}
}

// TestEnforceOpenKeepsUnknownKey checks that an instance with no token,
// which accepts a key it does not know, does not close that connection
// again when it enforces credentials after a tree change.
func TestEnforceOpenKeepsUnknownKey(t *testing.T) {
	a, _ := spikeAuthorizer(t, "")
	_, url, _ := spikeServer(t, a)

	stranger, _ := nkeys.CreateUser()
	nc, err := nats.Connect(url, append(nkeyOptions(stranger),
		nats.NoReconnect())...)
	if err != nil {
		t.Fatal("connect:", err)
	}
	defer nc.Close()

	a.enforce()

	if err := nc.Flush(); err != nil {
		t.Fatal("flush:", err)
	}
	time.Sleep(200 * time.Millisecond)
	if !nc.IsConnected() {
		t.Fatal("enforce closed an unknown key on an open instance")
	}
}

// TestEnforceKeepsConnBeingAccepted checks that enforcement leaves alone a
// connection the authorizer has not recorded yet. The server lists a
// connection with its key as soon as it has parsed CONNECT, while the
// authorizer is still verifying it, and enforcement running then used to
// close it as a key it knew nothing about. A device enrolling a second key
// saw its connection drop.
func TestEnforceKeepsConnBeingAccepted(t *testing.T) {
	a, _ := spikeAuthorizer(t, "tok")
	_, url, _ := spikeServer(t, a)

	// a key the instance does not know, enrolling with a live token
	const enrollToken = "enroll-secret"
	hash := client.HashEnrollToken(enrollToken)
	a.enroll[hash] = &enrollEntry{nodeID: "et"}
	a.enrollIDs["et"] = hash

	kp, _ := nkeys.CreateUser()
	nc, err := nats.Connect(url, append(nkeyOptions(kp),
		nats.Token(enrollToken), nats.NoReconnect())...)
	if err != nil {
		t.Fatal("connect:", err)
	}
	defer nc.Close()

	// the moment between the server listing the connection and the
	// authorizer recording it
	a.mu.Lock()
	saved := a.conns
	a.conns = make(map[uint64]authConn)
	a.mu.Unlock()

	a.enforce()

	if err := nc.Flush(); err != nil {
		t.Fatal("enforce closed a connection that was still being accepted:", err)
	}
	time.Sleep(200 * time.Millisecond)
	if !nc.IsConnected() {
		t.Fatal("enforce closed a connection that was still being accepted")
	}

	// once recorded, the connection is enforced as usual: revoking the
	// token it enrolled with closes it
	a.mu.Lock()
	a.conns = saved
	a.enroll[hash].disabled = true
	a.mu.Unlock()

	a.enforce()

	start := time.Now()
	for nc.IsConnected() {
		if time.Since(start) > 5*time.Second {
			t.Fatal("enforce kept a connection whose enrollment token was revoked")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestEnforceKeepsNewEntry checks that enforcement does not drop the entry
// for a connection the server did not list when enforcement read the
// connection list, which happens when the entry is written just after.
func TestEnforceKeepsNewEntry(t *testing.T) {
	a := newAuthorizer("tok", "")
	a.ready = true

	a.conns[1] = authConn{pubKey: "UNEW", enrollID: "et", added: time.Now()}
	a.conns[2] = authConn{pubKey: "UGONE",
		added: time.Now().Add(-2 * connRegisterGrace)}

	a.enforce()

	if _, ok := a.conns[1]; !ok {
		t.Error("dropped the entry for a connection still being accepted")
	}
	if _, ok := a.conns[2]; ok {
		t.Error("kept the entry for a connection that has gone")
	}
}

// TestDeviceAuthRequiredKeepsLocalToken starts a full server with device
// auth required and checks that its own client, which presents the shared
// token from loopback, still works.
func TestDeviceAuthRequiredKeepsLocalToken(t *testing.T) {
	opts := TestServerOptions
	opts.AuthToken = "tok"
	opts.DeviceAuth = DeviceAuthRequired

	nc, root, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	if root.ID == "" {
		t.Fatal("no root")
	}

	// a second local client with the token works too (the CLI case)
	nc2, err := nats.Connect(opts.NatsServer, nats.Token("tok"))
	if err != nil {
		t.Fatal("local token client refused:", err)
	}
	defer nc2.Close()

	if _, err := client.GetRootNode(nc2); err != nil {
		t.Fatal("local token client cannot read root:", err)
	}

	// and the wrong token does not
	if _, err := nats.Connect(opts.NatsServer, nats.Token("bad")); err == nil {
		t.Fatal("wrong token accepted")
	}

	_ = nc
}

// encodeSig encodes a signature the way nats.go sends it.
func encodeSig(sig []byte) []byte {
	return []byte(base64.RawURLEncoding.EncodeToString(sig))
}

// fakeUsers is a userAuthority for exercising checkUser without a store.
type fakeUsers struct {
	id      string
	expires time.Time
	anchors map[string][]string
}

const fakeJWT = "eyJhbGciOiJIUzI1NiJ9.valid.sig"

func (f fakeUsers) UserFromToken(token string) (string, time.Time, bool) {
	if token != fakeJWT {
		return "", time.Time{}, false
	}
	return f.id, f.expires, true
}

func (f fakeUsers) UserAnchors(id string) []string { return f.anchors[id] }
func (f fakeUsers) AuthAllowed(string) bool        { return true }
func (f fakeUsers) AuthFailed(string, string)      {}

func TestUserPermissions(t *testing.T) {
	p := userPermissions("U", []string{"G1", "G2"})

	wantPub := []string{"u.G1.U.>", "u.G2.U.>", "auth.me"}
	if !slices.Equal(p.Publish.Allow, wantPub) {
		t.Errorf("publish allow:\n got %v\nwant %v", p.Publish.Allow, wantPub)
	}

	wantSub := []string{"up.G1.>", "up.G2.>", "_INBOX_U.>"}
	if !slices.Equal(p.Subscribe.Allow, wantSub) {
		t.Errorf("subscribe allow:\n got %v\nwant %v", p.Subscribe.Allow, wantSub)
	}

	if p.Publish.Deny != nil || p.Subscribe.Deny != nil {
		t.Error("expected allow lists only")
	}

	for _, s := range append(p.Publish.Allow, p.Subscribe.Allow...) {
		for _, bad := range []string{"p.", "nodes.", "ep.", "$JS.", "auth.user", "admin.", "_INBOX."} {
			if strings.HasPrefix(s, bad) {
				t.Errorf("permission set includes %v", s)
			}
		}
	}
}

func TestCheckUser(t *testing.T) {
	exp := time.Now().Add(time.Hour).Truncate(time.Second)
	users := fakeUsers{id: "U", expires: exp, anchors: map[string][]string{"U": {"G"}}}

	tests := []struct {
		name     string
		token    string
		users    userAuthority
		user     string
		pass     string
		want     bool
		wantUser bool
	}{
		{"valid", "tok", users, "U", fakeJWT, true, true},
		{"valid on an open instance", "", users, "U", fakeJWT, true, true},
		{"issued to another user", "tok", users, "V", fakeJWT, false, false},
		{"invalid token", "tok", users, "U", "eyJ.bad.sig", false, false},
		{"invalid token on an open instance", "", users, "U", "eyJ.bad.sig", false, false},
		{"not in the tree", "tok", fakeUsers{id: "U", anchors: nil}, "U", fakeJWT, false, false},
		{"store not ready", "tok", nil, "U", fakeJWT, false, false},
		// an MQTT client sends user and password, and the server copies
		// the password into the token field: that is the shared token
		// path, not a user
		{"mqtt with the shared token", "tok", users, "mqtt", "tok", true, false},
		{"mqtt with a wrong token", "tok", users, "mqtt", "bad", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newAuthorizer(tt.token, "")
			if tt.users != nil {
				a.users = tt.users
			}
			c := &fakeClient{addr: tcpAddr("10.1.1.1:1")}
			c.opts.Username = tt.user
			c.opts.Password = tt.pass
			c.opts.Token = tt.pass
			if got := a.Check(c); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			if (c.user != nil) != tt.wantUser {
				t.Fatalf("user registered: %v, want %v", c.user != nil, tt.wantUser)
			}
			if !tt.wantUser {
				return
			}
			if c.user.Username != "U" {
				t.Errorf("username %v", c.user.Username)
			}
			if !c.user.ConnectionDeadline.Equal(exp) {
				t.Errorf("deadline %v, want %v", c.user.ConnectionDeadline, exp)
			}
			if !slices.Equal(c.user.Permissions.Publish.Allow, []string{"u.G.U.>", "auth.me"}) {
				t.Errorf("publish allow %v", c.user.Permissions.Publish.Allow)
			}
			if ac := a.conns[c.GetID()]; ac.userID != "U" || ac.grant != "G" {
				t.Errorf("connection not recorded: %+v", ac)
			}
		})
	}
}
