package client

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// fakeFuture is a resolved jetstream.PubAckFuture.
type fakeFuture struct {
	ok  chan *jetstream.PubAck
	err chan error
}

func (f fakeFuture) Ok() <-chan *jetstream.PubAck { return f.ok }
func (f fakeFuture) Err() <-chan error            { return f.err }
func (f fakeFuture) Msg() *nats.Msg               { return nil }

// fakePublisher accepts publishes and resolves each one immediately, failing
// the publish at failAt (1-based, 0 for none). rejectAt instead fails the
// PublishAsync call itself rather than the future.
type fakePublisher struct {
	failAt    int
	rejectAt  int
	n         int
	published []string
}

func (p *fakePublisher) PublishAsync(subject string, _ []byte,
	_ ...jetstream.PublishOpt) (jetstream.PubAckFuture, error) {

	p.n++
	if p.n == p.rejectAt {
		return nil, errors.New("stream not found")
	}

	p.published = append(p.published, subject)

	f := fakeFuture{
		ok:  make(chan *jetstream.PubAck, 1),
		err: make(chan error, 1),
	}
	if p.n == p.failAt {
		f.err <- errors.New("wrong last sequence")
	} else {
		f.ok <- &jetstream.PubAck{}
	}

	return f, nil
}

// fakeMsg is a source message that counts its acknowledgements.
type fakeMsg struct {
	subject string
	acked   *int
}

func (m fakeMsg) Subject() string { return m.subject }
func (m fakeMsg) Data() []byte    { return []byte("points") }
func (m fakeMsg) Ack() error      { *m.acked++; return nil }

func testWindow(n int, acked *int) []pumpMsg {
	window := make([]pumpMsg, n)
	for i := range window {
		window[i] = fakeMsg{subject: "inst.b.o.node.p.value." + string(rune('a'+i)), acked: acked}
	}
	return window
}

func TestSendWindowAcksAfterEveryPublishLands(t *testing.T) {
	acked := 0
	dst := &fakePublisher{}

	if err := sendWindow(context.Background(), dst, testWindow(5, &acked)); err != nil {
		t.Fatal("Error sending window:", err)
	}

	if acked != 5 {
		t.Fatalf("expected all 5 source messages acked, got %v", acked)
	}
	if len(dst.published) != 5 {
		t.Fatalf("expected 5 publishes, got %v", len(dst.published))
	}
}

// TestSendWindowAcksNothingWhenAPublishFails is the ordering guarantee: one
// rejected publish must leave the entire window unacknowledged, so the durable
// consumer redelivers all of it in source order. Acking the messages that did
// land would let the resend of the failed one arrive after messages stored
// later, and the receiving store reads the last message on a subject as that
// subject's current value.
func TestSendWindowAcksNothingWhenAPublishFails(t *testing.T) {
	acked := 0
	dst := &fakePublisher{failAt: 3}

	err := sendWindow(context.Background(), dst, testWindow(5, &acked))
	if err == nil {
		t.Fatal("expected an error when a publish in the window fails")
	}
	if !strings.Contains(err.Error(), "wrong last sequence") {
		t.Fatalf("error did not carry the publish failure: %v", err)
	}

	if acked != 0 {
		t.Fatalf("expected no source messages acked, got %v", acked)
	}
}

// A window is sent as a unit, so a failure part way through does not stop the
// remaining messages being offered; what it stops is the acknowledgement.
func TestSendWindowPublishesWholeWindowBeforeChecking(t *testing.T) {
	acked := 0
	dst := &fakePublisher{failAt: 2}

	if err := sendWindow(context.Background(), dst, testWindow(4, &acked)); err == nil {
		t.Fatal("expected an error when a publish in the window fails")
	}

	if len(dst.published) != 4 {
		t.Fatalf("expected the whole window published, got %v", len(dst.published))
	}
	if acked != 0 {
		t.Fatalf("expected no source messages acked, got %v", acked)
	}
}

// A publish the client refuses outright is handled the same way as one the
// server rejects: nothing is acknowledged.
func TestSendWindowAcksNothingWhenPublishIsRefused(t *testing.T) {
	acked := 0
	dst := &fakePublisher{rejectAt: 2}

	err := sendWindow(context.Background(), dst, testWindow(4, &acked))
	if err == nil {
		t.Fatal("expected an error when a publish is refused")
	}
	if !strings.Contains(err.Error(), "stream not found") {
		t.Fatalf("error did not carry the refusal: %v", err)
	}

	if acked != 0 {
		t.Fatalf("expected no source messages acked, got %v", acked)
	}
}

// A cancelled session must not acknowledge a window it never confirmed.
func TestSendWindowAcksNothingWhenCancelled(t *testing.T) {
	acked := 0

	// a publisher whose futures never resolve
	dst := publisherStub(func() jetstream.PubAckFuture {
		return fakeFuture{
			ok:  make(chan *jetstream.PubAck),
			err: make(chan error),
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := sendWindow(ctx, dst, testWindow(3, &acked)); err == nil {
		t.Fatal("expected an error when the session is cancelled")
	}

	if acked != 0 {
		t.Fatalf("expected no source messages acked, got %v", acked)
	}
}

// An empty window is a no-op rather than an error, so a pump that is caught up
// does not treat idleness as a failure.
func TestSendWindowEmpty(t *testing.T) {
	if err := sendWindow(context.Background(), &fakePublisher{}, nil); err != nil {
		t.Fatal("expected an empty window to be a no-op, got:", err)
	}
}

// publisherStub adapts a future factory to the asyncPublisher interface.
type publisherStub func() jetstream.PubAckFuture

func (p publisherStub) PublishAsync(_ string, _ []byte,
	_ ...jetstream.PublishOpt) (jetstream.PubAckFuture, error) {
	return p(), nil
}
