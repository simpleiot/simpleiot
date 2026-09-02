package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"
)

// This file is the Phase 0 spike for per-device credentials, kept as the
// canary for nats-server upgrades: it pins down that the embedded server
// routes every listener through CustomClientAuthentication, that the
// scoped permission set is enough for the sync pumps, that it refuses
// everything else, and that a connection can be found and closed by its
// key.

// spikeServer starts a bare NATS server with JetStream and the authorizer,
// with the index seeded by hand so no store is involved.
func spikeServer(t *testing.T, a *authorizer) (*server.Server, string, string) {
	t.Helper()

	dir, err := os.MkdirTemp("", "siot-auth-spike")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	opts := &server.Options{
		Host:                       "127.0.0.1",
		Port:                       -1,
		NoSigs:                     true,
		JetStream:                  true,
		StoreDir:                   dir,
		CustomClientAuthentication: a,
		AlwaysEnableNonce:          true,
	}
	opts.Websocket.Host = "127.0.0.1"
	opts.Websocket.Port = -1
	opts.Websocket.NoTLS = true

	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatal(err)
	}
	ns.Start()
	t.Cleanup(ns.Shutdown)
	a.ns = ns

	if !ns.ReadyForConnections(5 * time.Second) {
		t.Fatal("server not ready")
	}

	return ns, ns.ClientURL(), ns.WebsocketURL()
}

// spikeAuthorizer returns an authorizer that knows one device X with one
// credential, on an upstream with root R that has an origin stream for X.
func spikeAuthorizer(t *testing.T, token string) (*authorizer, nkeys.KeyPair) {
	t.Helper()

	kp, err := nkeys.CreateUser()
	if err != nil {
		t.Fatal(err)
	}
	pub, _ := kp.PublicKey()

	a := newAuthorizer(token, "")
	a.ready = true
	a.rootID = "R"
	a.creds[pub] = &credEntry{credID: "cred-x", deviceID: "X"}
	a.credIDs["cred-x"] = pub
	a.origins["X"] = map[string]bool{"R": true}

	return a, kp
}

func nkeyOption(kp nkeys.KeyPair) nats.Option {
	pub, _ := kp.PublicKey()
	return nats.Nkey(pub, kp.Sign)
}

// errCollector records async NATS errors so a test can assert on a
// permissions violation.
type errCollector struct {
	mu   sync.Mutex
	errs []string
}

func (c *errCollector) handler(_ *nats.Conn, _ *nats.Subscription, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errs = append(c.errs, err.Error())
}

func (c *errCollector) saw(substr string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.errs {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}

func TestAuthSpike(t *testing.T) {
	a, kp := spikeAuthorizer(t, "tok")
	ns, url, wsURL := spikeServer(t, a)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// ---- token client has full access and sets the stage as the upstream
	full, err := nats.Connect(url, nats.Token("tok"))
	if err != nil {
		t.Fatal("token connect:", err)
	}
	defer full.Close()

	jsFull, _ := jetstream.New(full)
	for _, s := range []struct{ b, o string }{{"X", "R"}, {"Y", "Y"}} {
		_, err = jsFull.CreateStream(ctx, jetstream.StreamConfig{
			Name:     fmt.Sprintf("inst_%v_%v", s.b, s.o),
			Subjects: []string{fmt.Sprintf("inst.%v.%v.>", s.b, s.o)},
		})
		if err != nil {
			t.Fatal("create stream:", err)
		}
	}
	for i := range 3 {
		if _, err := jsFull.Publish(ctx, "inst.X.R.n1.p.description.0",
			[]byte(fmt.Sprint(i))); err != nil {
			t.Fatal("publish:", err)
		}
	}

	// ---- wrong token and unknown key are refused
	if _, err := nats.Connect(url, nats.Token("bad")); !errors.Is(err, nats.ErrAuthorization) {
		t.Fatal("expected bad token to be refused, got:", err)
	}
	stranger, _ := nkeys.CreateUser()
	if _, err := nats.Connect(url, nkeyOption(stranger)); !errors.Is(err, nats.ErrAuthorization) {
		t.Fatal("expected unknown key to be refused, got:", err)
	}

	// ---- device X connects with its key; the subjects sync needs work
	var errs errCollector
	dev, err := nats.Connect(url, nkeyOption(kp), nats.NoReconnect(),
		nats.ErrorHandler(errs.handler))
	if err != nil {
		t.Fatal("device connect:", err)
	}
	defer dev.Close()

	jsDev, _ := jetstream.New(dev)

	// push side: create own replica stream and write into it
	if _, err := jsDev.Stream(ctx, "inst_X_X"); !errors.Is(err, jetstream.ErrStreamNotFound) {
		t.Fatal("expected stream not found, got:", err)
	}
	if _, err := jsDev.CreateStream(ctx, jetstream.StreamConfig{
		Name: "inst_X_X", Subjects: []string{"inst.X.X.>"},
	}); err != nil {
		t.Fatal("device create own stream:", err)
	}
	if _, err := jsDev.Publish(ctx, "inst.X.X.n1.p.value.0", []byte("1")); err != nil {
		t.Fatal("device publish own stream:", err)
	}

	// discovery: names for the boundary
	var names []string
	lister := jsDev.StreamNames(ctx, jetstream.WithStreamListSubject("inst.X.>"))
	for n := range lister.Name() {
		names = append(names, n)
	}
	if err := lister.Err(); err != nil {
		t.Fatal("stream names:", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected inst_X_R and inst_X_X, got %v", names)
	}

	// pull side: durable consumer on the upstream origin stream
	src, err := jsDev.Stream(ctx, "inst_X_R")
	if err != nil {
		t.Fatal("device stream info:", err)
	}
	cons, err := src.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable: "sync-X", AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatal("device create consumer:", err)
	}
	it, err := cons.Messages(jetstream.PullMaxMessages(256))
	if err != nil {
		t.Fatal("device messages:", err)
	}
	for i := range 3 {
		m, err := it.Next()
		if err != nil {
			t.Fatalf("device next %v: %v", i, err)
		}
		if err := m.Ack(); err != nil {
			t.Fatalf("device ack %v: %v", i, err)
		}
	}
	it.Stop()

	// the plain requests sync makes
	for _, subj := range []string{"nodes.root.all", "nodes.all.X"} {
		_, err := dev.Request(subj, nil, 300*time.Millisecond)
		if !errors.Is(err, nats.ErrNoResponders) {
			t.Fatalf("request %v: expected no responders (nothing serves it here), got %v", subj, err)
		}
	}
	if errs.saw("Permissions Violation") {
		t.Fatalf("permissions violation during allowed operations: %v", errs.errs)
	}

	// ---- anything outside the boundary is refused and does not land.
	// A refused publish gets no ack, so each one waits on its own short
	// deadline.
	short := func() context.Context {
		c, cancel := context.WithTimeout(ctx, time.Second)
		t.Cleanup(cancel)
		return c
	}
	if _, err := jsDev.Publish(short(), "inst.Y.Y.n1.p.value.0", []byte("x"),
		jetstream.WithRetryAttempts(0)); err == nil {
		t.Fatal("expected publish into another boundary to fail")
	}
	if _, err := jsDev.Stream(short(), "inst_Y_Y"); err == nil {
		t.Fatal("expected stream info on another boundary to fail")
	}
	if _, err := dev.Request("nodes.all.Y", nil, 300*time.Millisecond); err == nil {
		t.Fatal("expected node request on another device to fail")
	}
	if _, err := dev.Request("admin.storeMaint", nil, 300*time.Millisecond); err == nil {
		t.Fatal("expected admin request to fail")
	}
	if _, err := dev.SubscribeSync("p.>"); err == nil {
		dev.Flush()
	}
	time.Sleep(200 * time.Millisecond)
	if !errs.saw("Permissions Violation") {
		t.Fatalf("expected a permissions violation, got %v", errs.errs)
	}

	sy, err := jsFull.Stream(ctx, "inst_Y_Y")
	if err != nil {
		t.Fatal(err)
	}
	info, err := sy.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if info.State.Msgs != 0 {
		t.Fatal("message landed in another boundary's stream")
	}

	// ---- the connection is visible by key, and can be closed by ID
	live := a.liveConns()
	pub, _ := kp.PublicKey()
	var cid uint64
	for id, key := range live {
		if key == pub {
			cid = id
		}
	}
	if cid == 0 {
		t.Fatalf("device connection not found in %v", live)
	}
	if err := ns.DisconnectClientByID(cid); err != nil {
		t.Fatal("disconnect:", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for dev.IsConnected() && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if dev.IsConnected() {
		t.Fatal("device still connected after DisconnectClientByID")
	}

	// ---- WebSocket goes through the same authorizer
	ws, err := nats.Connect(wsURL, nkeyOption(kp), nats.NoReconnect())
	if err != nil {
		t.Fatal("websocket connect:", err)
	}
	ws.Close()
	if _, err := nats.Connect(wsURL, nkeyOption(stranger)); !errors.Is(err, nats.ErrAuthorization) {
		t.Fatal("expected unknown key to be refused over websocket, got:", err)
	}
}

func TestAuthSpikeOpen(t *testing.T) {
	// no token configured: anything without a key is accepted, as today
	a, kp := spikeAuthorizer(t, "")
	_, url, _ := spikeServer(t, a)

	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatal("open connect:", err)
	}
	nc.Close()

	nc, err = nats.Connect(url, nats.Token("anything"))
	if err != nil {
		t.Fatal("open connect with token:", err)
	}
	nc.Close()

	// a key the instance does not know is accepted like anything else
	// (with full access), while a known one is scoped
	stranger, _ := nkeys.CreateUser()
	nc, err = nats.Connect(url, nkeyOption(stranger))
	if err != nil {
		t.Fatal("expected unknown key to be accepted on an open instance, got:", err)
	}
	if _, err := nc.SubscribeSync("p.>"); err != nil {
		t.Fatal(err)
	}
	nc.Close()

	var errs errCollector
	nc, err = nats.Connect(url, nkeyOption(kp), nats.ErrorHandler(errs.handler))
	if err != nil {
		t.Fatal("device connect:", err)
	}
	if _, err := nc.SubscribeSync("p.>"); err == nil {
		nc.Flush()
	}
	time.Sleep(200 * time.Millisecond)
	if !errs.saw("Permissions Violation") {
		t.Fatal("known key not scoped on an open instance")
	}
	nc.Close()
}
