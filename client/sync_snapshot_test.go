package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// newPumpServer starts a JetStream server for pump tests and returns a
// JetStream context on a full-access connection. users, when set, are the
// server's only accounts.
func newPumpServer(t *testing.T, users []*server.User) (*server.Server, jetstream.JetStream) {
	t.Helper()

	dir, err := os.MkdirTemp("", "siot-pump-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	ns, err := server.NewServer(&server.Options{
		Port:      -1,
		JetStream: true,
		StoreDir:  dir,
		NoSigs:    true,
		Users:     users,
	})
	if err != nil {
		t.Fatal(err)
	}
	ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		t.Fatal("NATS server failed to start")
	}
	t.Cleanup(ns.Shutdown)

	var opts []nats.Option
	if len(users) > 0 {
		opts = append(opts, nats.UserInfo(users[0].Username, users[0].Password))
	}
	nc, err := nats.Connect(ns.ClientURL(), opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatal(err)
	}
	return ns, js
}

// snapRecorder collects snapshot messages as they arrive.
type snapRecorder struct {
	mu   sync.Mutex
	msgs map[string]string
	n    int
}

func recordSnapshots(t *testing.T, nc *nats.Conn) *snapRecorder {
	t.Helper()
	r := &snapRecorder{msgs: make(map[string]string)}
	sub, err := nc.Subscribe(SnapshotPrefix+".>", func(m *nats.Msg) {
		r.mu.Lock()
		r.msgs[m.Subject] = string(m.Data)
		r.n++
		r.mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sub.Unsubscribe() })
	if err := nc.Flush(); err != nil {
		t.Fatal(err)
	}
	return r
}

func (r *snapRecorder) get(subject string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.msgs[subject]
	return v, ok
}

func (r *snapRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.n
}

func pumpWait(t *testing.T, what string, cond func() bool) {
	t.Helper()
	start := time.Now()
	for !cond() {
		if time.Since(start) > 15*time.Second {
			t.Fatal("timeout waiting for", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func streamMsgs(t *testing.T, js jetstream.JetStream, name string) uint64 {
	t.Helper()
	s, err := js.Stream(context.Background(), name)
	if err != nil {
		return 0
	}
	info, err := s.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return info.State.Msgs
}

func slowPump(t *testing.T, delay time.Duration) {
	t.Helper()
	pumpWindowDelay = delay
	snapshotInactive = time.Second
	t.Cleanup(func() {
		pumpWindowDelay = 0
		snapshotInactive = 30 * time.Second
	})
}

const (
	pumpTestName = "inst_b_o"
	pumpTestRPM  = "inst.b.o.n1.p.rpm.0"
	pumpTestTemp = "inst.b.o.n1.p.temp.0"
	pumpTestMode = "inst.b.o.n1.p.mode.0"
)

func newPumpSource(t *testing.T) jetstream.JetStream {
	t.Helper()
	_, js := newPumpServer(t, nil)
	_, err := js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     pumpTestName,
		Subjects: []string{"inst.b.o.>"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return js
}

func pumpPublish(t *testing.T, js jetstream.JetStream, subject, payload string) uint64 {
	t.Helper()
	ack, err := js.Publish(context.Background(), subject, []byte(payload))
	if err != nil {
		t.Fatal(err)
	}
	return ack.Sequence
}

// TestPumpSnapshot covers a reconnect after time offline: the current value
// of each subject that changed is sent ahead of the backlog, a subject that
// did not change is not, a point written during catch-up is sent ahead of the
// history too, and the snapshot stops and its consumer goes away once the
// history has caught up.
func TestPumpSnapshot(t *testing.T) {
	slowPump(t, 400*time.Millisecond)
	ctx := context.Background()

	src := newPumpSource(t)
	_, dst := newPumpServer(t, nil)
	snaps := recordSnapshots(t, dst.Conn())

	// an earlier session sent rpm and the first temp
	pumpPublish(t, src, pumpTestRPM, "rpm-1")
	pumpPublish(t, src, pumpTestTemp, "temp-1")
	s, err := src.Stream(ctx, pumpTestName)
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "sync-up",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := c.Fetch(2)
	if err != nil {
		t.Fatal(err)
	}
	for m := range batch.Messages() {
		if err := m.DoubleAck(ctx); err != nil {
			t.Fatal(err)
		}
	}

	// the backlog written while offline
	const backlog = 1000
	for i := 2; i <= backlog; i++ {
		pumpPublish(t, src, pumpTestTemp, fmt.Sprintf("temp-%v", i))
	}
	last := pumpPublish(t, src, pumpTestMode, "mode-1")

	pctx, cancel := context.WithCancel(ctx)
	defer cancel()
	p, err := runPump(pctx, src, dst, "test", "b", "o", "up")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	pumpWait(t, "current values", func() bool {
		_, temp := snaps.get(SubjectSnapshot(pumpTestTemp))
		_, mode := snaps.get(SubjectSnapshot(pumpTestMode))
		return temp && mode
	})
	if got := streamMsgs(t, dst, pumpTestName); got >= last-2 {
		t.Fatalf("history already caught up (%v messages); the test proves nothing", got)
	}
	if v, _ := snaps.get(SubjectSnapshot(pumpTestTemp)); v != fmt.Sprintf("temp-%v", backlog) {
		t.Errorf("temp snapshot is %q, expected the newest value", v)
	}
	if _, ok := snaps.get(SubjectSnapshot(pumpTestRPM)); ok {
		t.Error("rpm did not change since the last session but was sent")
	}

	// a point written during catch-up goes ahead of the history
	newest := pumpPublish(t, src, pumpTestTemp, "temp-new")
	pumpWait(t, "point written during catch-up", func() bool {
		v, _ := snaps.get(SubjectSnapshot(pumpTestTemp))
		return v == "temp-new"
	})
	if got := streamMsgs(t, dst, pumpTestName); got >= newest-2 {
		t.Fatalf("history reached the new point (%v messages) before the snapshot", got)
	}

	pumpWait(t, "history to catch up", func() bool {
		return streamMsgs(t, dst, pumpTestName) == newest-2
	})

	// the snapshot consumer expires once the history has caught up
	pumpWait(t, "snapshot consumer to go away", func() bool {
		names := s.ConsumerNames(ctx)
		n := 0
		for range names.Name() {
			n++
		}
		return n == 1
	})

	// once caught up, the history alone carries new points
	sent := snaps.count()
	after := pumpPublish(t, src, pumpTestTemp, "temp-after")
	pumpWait(t, "point after catch-up", func() bool {
		return streamMsgs(t, dst, pumpTestName) == after-2
	})
	time.Sleep(200 * time.Millisecond)
	if snaps.count() != sent {
		t.Errorf("snapshot sent %v messages after catch-up", snaps.count()-sent)
	}
}

// TestPumpSnapshotFirstSync checks that a first sync, with no durable
// consumer yet, sends every subject's current value ahead of the history.
func TestPumpSnapshotFirstSync(t *testing.T) {
	slowPump(t, 200*time.Millisecond)

	src := newPumpSource(t)
	_, dst := newPumpServer(t, nil)
	snaps := recordSnapshots(t, dst.Conn())

	pumpPublish(t, src, pumpTestRPM, "rpm-1")
	pumpPublish(t, src, pumpTestMode, "mode-1")
	for i := 1; i <= 600; i++ {
		pumpPublish(t, src, pumpTestTemp, fmt.Sprintf("temp-%v", i))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p, err := runPump(ctx, src, dst, "test", "b", "o", "up")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	pumpWait(t, "every subject's current value", func() bool {
		rpm, _ := snaps.get(SubjectSnapshot(pumpTestRPM))
		mode, _ := snaps.get(SubjectSnapshot(pumpTestMode))
		temp, _ := snaps.get(SubjectSnapshot(pumpTestTemp))
		return rpm == "rpm-1" && mode == "mode-1" && temp == "temp-600"
	})

	pumpWait(t, "history to catch up", func() bool {
		return streamMsgs(t, dst, pumpTestName) == 602
	})
}

// TestPumpSnapshotRefused covers an upstream from before snapshots, whose
// device grant has no snap subject: the snapshot stops after the first
// refused message and the history still catches up.
func TestPumpSnapshotRefused(t *testing.T) {
	slowPump(t, 0)

	src := newPumpSource(t)
	_, dst := newPumpServer(t, []*server.User{{
		Username: "device",
		Password: "pw",
		Permissions: &server.Permissions{
			Publish: &server.SubjectPermission{
				Allow: []string{"inst.>", "$JS.>", "_INBOX.>"},
			},
			Subscribe: &server.SubjectPermission{Allow: []string{"_INBOX.>"}},
		},
	}})

	for i := 0; i < 50; i++ {
		pumpPublish(t, src, fmt.Sprintf("inst.b.o.n%v.p.value.0", i), "v")
	}

	s, err := src.Stream(context.Background(), pumpTestName)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sent, err := runSnapshot(ctx, s, dst.Conn(), "inst.b.o.>", 0, time.Second)
	if !errors.Is(err, errSnapshotRefused) {
		t.Fatalf("snapshot returned %v, expected it to be refused", err)
	}
	if sent != 1 {
		t.Errorf("snapshot sent %v messages, expected to stop after the first", sent)
	}

	pctx, pcancel := context.WithCancel(context.Background())
	defer pcancel()
	p, err := runPump(pctx, src, dst, "test", "b", "o", "up")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	pumpWait(t, "history to catch up", func() bool {
		return streamMsgs(t, dst, pumpTestName) == 50
	})
}
