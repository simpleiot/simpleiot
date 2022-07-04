package client_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
	"github.com/simpleiot/simpleiot/store"
)

type testNode struct {
	Description string `point:"description"`
	Port        int    `point:"port"`
}

type testNodeClient struct {
	config testNode
}

func newTestNodeClient(config testNode) client.Client {
	return &testNodeClient{config: config}
}

func (tnc *testNodeClient) run() {
	fmt.Println("Test to see other functions can be called")
}

func (tnc *testNodeClient) Run(c <-chan data.Point) {
	tnc.run()
}

func (tnc *testNodeClient) Stop() {
}

var testServerOptions = server.Options{
	StoreType:    store.StoreTypeMemory,
	NatsPort:     4990,
	HTTPPort:     "8990",
	NatsHTTPPort: 8991,
	NatsWSPort:   8992,
	NatsServer:   "nats://localhost:4990",
}

func TestManager(t *testing.T) {
	s, nc, err := server.NewServer(testServerOptions)

	if err != nil {
		t.Fatal("Error starting siot server: ", err)
	}

	stopped := make(chan struct{})

	go func() {
		err := s.Start()
		log.Println("Server start returned: ", err)
		close(stopped)
	}()

	defer func() {
		s.Stop(nil)
		<-stopped
	}()

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	err = s.WaitStart(ctx)

	if err != nil {
		t.Fatal("Error waiting for server to start: ", err)
	}

	nodes, err := client.GetNode(nc, "root", "")

	if err != nil {
		t.Fatal("Error getting root node: ", err)
	}

	log.Println("CLIFF: rootnodes: ", nodes)
}

func TestManager2(t *testing.T) {
	/*
		s, nc, err := test.StartServer()
		defer s.Stop()

		if err != nil {
			t.Fatal("Test server failed to start: ", err)
		}

		nodes, err := client.GetNode(nc, "root", "")

		if err != nil {
			t.Fatal("Error getting root node: ", err)
		}

		log.Println("CLIFF: rootnodes: ", nodes)

		m := client.NewManager("rootid", newTestNodeClient)
		_ = m
	*/
}
