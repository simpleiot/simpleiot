package server

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// TestServerOptions options used for test server
var TestServerOptions = Options{
	StoreFile:    "test.sqlite",
	NatsPort:     8900,
	HTTPPort:     "8901",
	NatsHTTPPort: 8902,
	NatsWSPort:   8903,
	NatsServer:   "nats://localhost:8900",
}

// TestServerOptions2 options used for 2nd test server
var TestServerOptions2 = Options{
	StoreFile:    "test2.sqlite",
	NatsPort:     8910,
	HTTPPort:     "8911",
	NatsHTTPPort: 8912,
	NatsWSPort:   8913,
	NatsServer:   "nats://localhost:8910",
}

// TestServer starts a test server and returns a function to stop it
func TestServer(args ...string) (*nats.Conn, data.NodeEdge, func(), error) {
	opts := TestServerOptions

	if len(args) > 0 {
		opts = TestServerOptions2
	}

	cleanup := func() {
		exec.Command("sh", "-c",
			fmt.Sprintf("rm %v*", opts.StoreFile)).Run()
	}

	cleanup()

	s, nc, err := NewServer(opts)

	if err != nil {
		return nil, data.NodeEdge{}, nil, fmt.Errorf("Error starting siot server: %v", err)
	}

	clients, err := client.DefaultClients(nc)
	s.AddClient(clients)

	stopped := make(chan struct{})

	go func() {
		err := s.Start()
		if err != nil {
			log.Println("Test Server start returned: ", err)
		}
		close(stopped)
	}()

	stop := func() {
		s.Stop(nil)
		<-stopped
		cleanup()
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	err = s.WaitStart(ctx)
	cancel()
	if err != nil {
		return nil, data.NodeEdge{}, stop, fmt.Errorf("Error waiting for test server to start: %v", err)
	}

	nodes, err := client.GetNodes(nc, "root", "all", "", false)

	if err != nil {
		return nil, data.NodeEdge{}, stop, fmt.Errorf("Get root nodes error: %v", err)
	}

	if len(nodes) < 1 {
		return nil, data.NodeEdge{}, stop, fmt.Errorf("Did not get a root node")
	}

	return nc, nodes[0], stop, nil
}
