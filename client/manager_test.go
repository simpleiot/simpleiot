package client

import (
	"fmt"
	"testing"

	"github.com/simpleiot/simpleiot/data"
)

type testNode struct {
	Description string `point:"description"`
	Port        int    `point:"port"`
}

type testNodeClient struct {
	config testNode
}

func newTestNodeClient(config testNode) Client {
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
	// import cycle not allowed ...
	// need to just instantiate the store only part
	//nc, err := simpleiot.Start()

	m := NewManager("rootid", newTestNodeClient)
	_ = m
}
