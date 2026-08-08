package store

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// TestJetStreamSourcingOverLeafnode is the ADR-7 Stage 3 prerequisite
// spike: verify a hub can hold a replica of a device-origin stream via
// JetStream sourcing across a leaf connection with distinct JetStream
// domains, and that the replica catches up after the device restarts.
func TestJetStreamSourcingOverLeafnode(t *testing.T) {
	hubDir, err := os.MkdirTemp("", "siot-hub-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(hubDir) }()
	devDir, err := os.MkdirTemp("", "siot-dev-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(devDir) }()

	// hub: JetStream domain "hub", leafnode listener
	hubOpts := &server.Options{
		Port:            -1,
		JetStream:       true,
		StoreDir:        hubDir,
		NoSigs:          true,
		JetStreamDomain: "hub",
	}
	hubOpts.LeafNode.Host = "127.0.0.1"
	hubOpts.LeafNode.Port = 17433

	hub, err := server.NewServer(hubOpts)
	if err != nil {
		t.Fatal("Error creating hub server:", err)
	}
	hub.Start()
	defer hub.Shutdown()
	if !hub.ReadyForConnections(5 * time.Second) {
		t.Fatal("hub failed to start")
	}

	leafURL, err := url.Parse("nats-leaf://127.0.0.1:17433")
	if err != nil {
		t.Fatal(err)
	}

	// device: JetStream domain "dev", leaf connection to the hub
	startDev := func() *server.Server {
		devOpts := &server.Options{
			Port:            -1,
			JetStream:       true,
			StoreDir:        devDir,
			NoSigs:          true,
			JetStreamDomain: "dev",
		}
		devOpts.LeafNode.Remotes = []*server.RemoteLeafOpts{
			{URLs: []*url.URL{leafURL}},
		}
		dev, err := server.NewServer(devOpts)
		if err != nil {
			t.Fatal("Error creating device server:", err)
		}
		dev.Start()
		if !dev.ReadyForConnections(5 * time.Second) {
			t.Fatal("device failed to start")
		}
		return dev
	}

	dev := startDev()

	ctx := context.Background()

	// device owns stream inst_X_X
	ncDev, err := nats.Connect(dev.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	jsDev, err := jetstream.New(ncDev)
	if err != nil {
		t.Fatal(err)
	}
	_, err = jsDev.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "inst_X_X",
		Subjects: []string{"inst.X.X.>"},
	})
	if err != nil {
		t.Fatal("Error creating device stream:", err)
	}

	publish := func(js jetstream.JetStream, n int) {
		for i := range n {
			_, err := js.Publish(ctx, "inst.X.X.n1.p.value.0",
				fmt.Appendf(nil, "v%v", i))
			if err != nil {
				t.Fatal("publish error:", err)
			}
		}
	}

	publish(jsDev, 3)

	// hub: replica of inst_X_X sourced from the device domain
	ncHub, err := nats.Connect(hub.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	defer ncHub.Close()
	jsHub, err := jetstream.New(ncHub)
	if err != nil {
		t.Fatal(err)
	}
	replica, err := jsHub.CreateStream(ctx, jetstream.StreamConfig{
		Name: "inst_X_X",
		Sources: []*jetstream.StreamSource{
			{Name: "inst_X_X", Domain: "dev"},
		},
	})
	if err != nil {
		t.Fatal("Error creating hub replica stream:", err)
	}

	waitMsgs := func(want uint64) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			info, err := replica.Info(ctx)
			if err == nil && info.State.Msgs >= want {
				if info.State.Msgs != want {
					t.Fatalf("replica has %v msgs, want %v", info.State.Msgs, want)
				}
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		info, _ := replica.Info(ctx)
		t.Fatalf("timeout: replica has %v msgs, want %v", info.State.Msgs, want)
	}

	// initial replication
	waitMsgs(3)

	// live replication
	publish(jsDev, 2)
	waitMsgs(5)

	// catch-up after a device restart: write while the link is down,
	// then verify only the missed messages arrive
	ncDev.Close()
	dev.Shutdown()
	dev.WaitForShutdown()

	dev = startDev()
	defer dev.Shutdown()

	ncDev, err = nats.Connect(dev.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	defer ncDev.Close()
	jsDev, err = jetstream.New(ncDev)
	if err != nil {
		t.Fatal(err)
	}

	publish(jsDev, 2)
	waitMsgs(7)
}
