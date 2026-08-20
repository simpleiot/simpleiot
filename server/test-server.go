package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// TestServerOptions options used for test server
var TestServerOptions = Options{
	NatsPort:     8900,
	HTTPPort:     "8901",
	NatsHTTPPort: 8902,
	NatsWSPort:   8903,
	NatsMQTTPort: 8904,
	NatsServer:   "nats://localhost:8900",
	ID:           "inst1",
}

// TestServerOptions2 options used for 2nd test server
var TestServerOptions2 = Options{
	NatsPort:     8910,
	HTTPPort:     "8911",
	NatsHTTPPort: 8912,
	NatsWSPort:   8913,
	NatsMQTTPort: 8914,
	NatsServer:   "nats://localhost:8910",
	ID:           "inst2",
}

// TestServer starts a test server and returns a function to stop it
func TestServer(args ...string) (*nats.Conn, data.NodeEdge, func(), error) {
	opts := TestServerOptions

	if len(args) > 0 {
		opts = TestServerOptions2
	}

	return TestServerOpts(opts)
}

// TestServerOpts starts a test server with the options given, which is how a
// test starts one with a provisioning directory. The store is cleaned out
// first, so the server comes up as a fresh instance.
func TestServerOpts(opts Options) (*nats.Conn, data.NodeEdge, func(), error) {
	return startTestServer(opts, true)
}

// TestServerOptsKeepStore starts a test server on whatever is already in the
// store, which is how a test restarts an instance rather than replacing it.
func TestServerOptsKeepStore(opts Options) (*nats.Conn, data.NodeEdge, func(), error) {
	return startTestServer(opts, false)
}

func startTestServer(opts Options, clean bool) (*nats.Conn, data.NodeEdge, func(), error) {
	// JetStream data lives in a directory named for the instance, so a
	// restart with TestServerOptsKeepStore finds the same store.
	if opts.DataDir == "" {
		opts.DataDir = filepath.Join(os.TempDir(), "siot-test-"+opts.ID)
	}

	cleanup := func() {
		_ = os.RemoveAll(opts.DataDir)
	}

	if clean {
		cleanup()
	}

	if err := os.MkdirAll(opts.DataDir, 0755); err != nil {
		return nil, data.NodeEdge{}, nil, fmt.Errorf("error creating data dir: %v", err)
	}

	s, nc, err := NewServer(opts)

	if err != nil {
		return nil, data.NodeEdge{}, nil, fmt.Errorf("error starting siot server: %v", err)
	}

	clients, _ := client.DefaultClients(nc)
	s.AddClient(clients)

	stopped := make(chan struct{})

	go func() {
		err := s.Run()
		if err != nil {
			log.Println("Test Server start returned:", err)
		}
		close(stopped)
	}()

	// the store is left in place on stop so a test can restart the instance
	// with TestServerOptsKeepStore; the next clean start removes it
	stop := func() {
		s.Stop(nil)
		<-stopped
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	err = s.WaitStart(ctx)
	cancel()
	if err != nil {
		return nil, data.NodeEdge{}, stop, fmt.Errorf("error waiting for test server to start: %v", err)
	}

	// Retry getting root nodes — store subscriptions may take a moment
	// to become active after the run group starts
	var nodes []data.NodeEdge
	for range 50 {
		nodes, err = client.GetNodes(nc, "root", "all", "", false)
		if err == nil && len(nodes) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err != nil {
		return nil, data.NodeEdge{}, stop, fmt.Errorf("get root nodes error: %v", err)
	}

	if len(nodes) < 1 {
		return nil, data.NodeEdge{}, stop, fmt.Errorf("did not get a root node")
	}

	return nc, nodes[0], stop, nil
}
