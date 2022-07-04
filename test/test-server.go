package test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
	"github.com/simpleiot/simpleiot/store"
)

var testServerOptions = server.Options{
	StoreType:    store.StoreTypeMemory,
	NatsPort:     4990,
	HTTPPort:     "8990",
	NatsHTTPPort: 8991,
	NatsWSPort:   8992,
	NatsServer:   "nats://localhost:4990",
}

// Server starts a test server and returns a function to stop it
func Server() (*nats.Conn, data.NodeEdge, func(), error) {
	s, nc, err := server.NewServer(testServerOptions)

	if err != nil {
		return nil, data.NodeEdge{}, nil, fmt.Errorf("Error starting siot server: %v", err)
	}

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
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	err = s.WaitStart(ctx)
	cancel()
	if err != nil {
		return nil, data.NodeEdge{}, stop, fmt.Errorf("Error waiting for test server to start: %v", err)
	}

	nodes, err := client.GetNode(nc, "root", "")

	if err != nil {
		return nil, data.NodeEdge{}, stop, fmt.Errorf("Get root nodes error: %v", err)
	}

	if len(nodes) < 1 {
		return nil, data.NodeEdge{}, stop, fmt.Errorf("Did not get a root node")
	}

	return nc, nodes[0], stop, nil
}
