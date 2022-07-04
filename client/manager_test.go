package client_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/test"
)

type testNode struct {
	Description string `point:"description"`
	Port        int    `point:"port"`
}

type testNodeClient struct {
	nc     *nats.Conn
	config testNode
}

func newTestNodeClient(nc *nats.Conn, config testNode) client.Client {
	return &testNodeClient{nc: nc, config: config}
}

func (tnc *testNodeClient) run() {
	fmt.Printf("tnc client run: %+v\n", tnc)
}

func (tnc *testNodeClient) Run(c <-chan data.Point) {
	tnc.run()
}

func (tnc *testNodeClient) Stop() {
}

func TestManager(t *testing.T) {
	nc, root, stop, err := test.Server()
	_ = nc

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	log.Println("CLIFF: rootnode: ", root)

	ne, err := data.Encode(testNode{"fancy test node", 8080})
	if err != nil {
		t.Fatal("Error encoding node: ", err)
	}

	ne.Parent = root.ID

	log.Printf("CLIFF: %+v\n", ne)

	// hydrate database with test data
	err = client.SendNode(nc, ne)

	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	m := client.NewManager[testNode](nc, root.ID, newTestNodeClient)

	err = m.Start()
	if err != nil {
		t.Fatal("Error starting manager: ", err)
	}
}

func TestManager2(t *testing.T) {
	nc, root, stop, err := test.Server()
	_ = nc

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	log.Println("CLIFF: rootnode: ", root)
}
