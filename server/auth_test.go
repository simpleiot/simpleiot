package server

import (
	"crypto/tls"
	"encoding/base64"
	"net"
	"slices"
	"testing"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/simpleiot/simpleiot/client"
)

func TestDevicePermissions(t *testing.T) {
	p := devicePermissions("X", "R", []string{"R", "R2"})

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

	if !slices.Equal(p.Subscribe.Allow, []string{"_INBOX.>"}) {
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
	p = devicePermissions("X", "R", []string{"X", "R"})
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
