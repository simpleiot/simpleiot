package client_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/test"
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

func TestManager(t *testing.T) {
	nc, stop, err := test.Server()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	nodes, err := client.GetNode(nc, "root", "")

	if err != nil {
		t.Fatal("Error getting root node: ", err)
	}

	log.Println("CLIFF: rootnodes: ", nodes)
}

func TestManager2(t *testing.T) {
	nc, stop, err := test.Server()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	nodes, err := client.GetNode(nc, "root", "")

	if err != nil {
		t.Fatal("Error getting root node: ", err)
	}

	log.Println("CLIFF: rootnodes: ", nodes)
}
