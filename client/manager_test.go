package client_test

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/test"
)

type testNode struct {
	ID          string `node:"id"`
	Description string `point:"description"`
	Port        int    `point:"port"`
}

type testNodeClient struct {
	nc          *nats.Conn
	config      testNode
	stop        chan struct{}
	stopped     chan struct{}
	newPoints   chan []data.Point
	chGetConfig chan chan testNode
}

func newTestNodeClient(nc *nats.Conn, config testNode) *testNodeClient {
	return &testNodeClient{
		nc:          nc,
		config:      config,
		stop:        make(chan struct{}),
		stopped:     make(chan struct{}),
		newPoints:   make(chan []data.Point),
		chGetConfig: make(chan chan testNode),
	}
}

func (tnc *testNodeClient) Start() error {
	for {
		select {
		case <-tnc.stop:
			close(tnc.stopped)
			return nil
		case pts := <-tnc.newPoints:
			err := data.MergePoints(pts, &tnc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
		case ch := <-tnc.chGetConfig:
			ch <- tnc.config
		}
	}

	return nil
}

func (tnc *testNodeClient) Stop(err error) {
	close(tnc.stop)
}

func (tnc *testNodeClient) Update(points []data.Point) {
	tnc.newPoints <- points
}

func (tnc *testNodeClient) getConfig() testNode {
	result := make(chan testNode)
	tnc.chGetConfig <- result
	return <-result
}

func TestManager(t *testing.T) {
	nc, root, stop, err := test.Server()
	_ = nc

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	testConfig := testNode{"", "fancy test node", 8080}

	ne, err := data.Encode(testConfig)
	if err != nil {
		t.Fatal("Error encoding node: ", err)
	}

	ne.Parent = root.ID

	// hydrate database with test data
	err = client.SendNode(nc, ne)

	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	var testClient *testNodeClient
	var testClientLock sync.Mutex

	// wrap newTestNodeClient so we can stash a link to testClient
	var newTestNodeClientWrapper = func(nc *nats.Conn, config testNode) client.Client {
		testClientLock.Lock()
		defer testClientLock.Unlock()
		testClient = newTestNodeClient(nc, config)
		return testClient
	}

	// Create a new manager for nodes of type "testNode". The manager looks for new nodes under the
	// root and if it finds any, it instantiates a new client, and sends point updates to it
	m := client.NewManager(nc, root.ID, newTestNodeClientWrapper)

	managerStopped := make(chan struct{})

	startErr := make(chan error)

	go func() {
		err = m.Start()
		if err != nil {
			startErr <- fmt.Errorf("manager start returned error: %v", err)
		}

		close(managerStopped)
	}()

	// wait for client to be created
	start := time.Now()
	for time.Since(start) < time.Second {
		testClientLock.Lock()
		if testClient != nil {
			testClientLock.Unlock()
			break
		}
		testClientLock.Unlock()
	}

	if testClient == nil {
		t.Fatal("Test client not created")
	}

	// verify config got passed in to the constructer
	currentConfig := testClient.getConfig()
	// ID was not populated when we originally created the node
	testConfig.ID = currentConfig.ID
	if currentConfig != testConfig {
		t.Errorf("Initial test config is not correct, exp %+v, got %+v", testConfig, currentConfig)
	}

	// Test point updates
	modifiedDescription := "updated description"

	err = client.SendNodePoint(nc, currentConfig.ID,
		data.Point{Type: "description", Text: modifiedDescription}, true)

	time.Sleep(10 * time.Millisecond)

	if testClient.getConfig().Description != modifiedDescription {
		t.Error("Description not modified")
	}

	// Shutdown
	m.Stop(nil)

	select {
	case <-testClient.stopped:
		// all is well
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for client to be stopped")
	}

	select {
	case <-managerStopped:
		// all is well
	case <-time.After(time.Second):
		t.Fatal("manager did not stop")
	}

	select {
	case <-startErr:
		t.Fatal("Manager start returned an error: ", err)
	default:
		// all is well
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
