package client_test

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/test"
)

type testNode struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Port        int    `point:"port"`
	Role        string `edgepoint:"role"`
}

type testNodeClient struct {
	nc            *nats.Conn
	config        testNode
	stop          chan struct{}
	stopped       chan struct{}
	newPoints     chan []data.Point
	newEdgePoints chan []data.Point
	chGetConfig   chan chan testNode
}

func newTestNodeClient(nc *nats.Conn, config testNode) *testNodeClient {
	return &testNodeClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		stopped:       make(chan struct{}),
		newPoints:     make(chan []data.Point),
		newEdgePoints: make(chan []data.Point),
		chGetConfig:   make(chan chan testNode),
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
		case pts := <-tnc.newEdgePoints:
			err := data.MergeEdgePoints(pts, &tnc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
		case ch := <-tnc.chGetConfig:
			ch <- tnc.config
		}
	}
}

func (tnc *testNodeClient) Stop(err error) {
	close(tnc.stop)
}

func (tnc *testNodeClient) Points(points []data.Point) {
	tnc.newPoints <- points
}

func (tnc *testNodeClient) EdgePoints(points []data.Point) {
	tnc.newEdgePoints <- points
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

	testConfig := testNode{"", "", "fancy test node", 8080, "admin"}

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
	newClient := make(chan *testNodeClient)

	// wrap newTestNodeClient so we can stash a link to testClient
	var newTestNodeClientWrapper = func(nc *nats.Conn, config testNode) client.Client {
		testClient := newTestNodeClient(nc, config)
		newClient <- testClient
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
	select {
	case testClient = <-newClient:
	case <-time.After(time.Second):
		t.Fatal("Test client not created")
	}

	// verify config got passed in to the constructer
	currentConfig := testClient.getConfig()
	// ID was not populated when we originally created the node
	testConfig.ID = currentConfig.ID
	testConfig.Parent = currentConfig.Parent
	if currentConfig != testConfig {
		t.Errorf("Initial test config is not correct, exp %+v, got %+v", testConfig, currentConfig)
	}

	// Test point updates
	modifiedDescription := "updated description"

	err = client.SendNodePoint(nc, currentConfig.ID,
		data.Point{Type: "description", Text: modifiedDescription}, true)

	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if testClient.getConfig().Description != modifiedDescription {
		t.Error("Description not modified")
	}

	// Test edge point updates
	modifiedRole := "user"

	err = client.SendEdgePoint(nc, currentConfig.ID, currentConfig.Parent,
		data.Point{Type: "role", Text: modifiedRole}, true)

	if err != nil {
		t.Errorf("Error sending edge point: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if testClient.getConfig().Role != modifiedRole {
		t.Error("Role not modified")
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

func TestManagerAddRemove(t *testing.T) {
	nc, root, stop, err := test.Server()
	_ = nc

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	newClient := make(chan *testNodeClient)

	// wrap newTestNodeClient so we can get a handle to new clients
	var newTestNodeClientWrapper = func(nc *nats.Conn, config testNode) client.Client {
		testClient := newTestNodeClient(nc, config)
		newClient <- testClient
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

	// populate with new testNode
	testConfig := testNode{"", "", "fancy test node", 8080, "admin"}

	ne, err := data.Encode(testConfig)
	if err != nil {
		t.Fatal("Error encoding node: ", err)
	}

	ne.Parent = root.ID

	// populate database with test node
	err = client.SendNode(nc, ne)

	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	var testClient *testNodeClient

	// wait for client to be created
	select {
	case testClient = <-newClient:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting for new client to be created")
	}

	// verify config got passed in to the constructer
	currentConfig := testClient.getConfig()
	// ID was not populated when we originally created the node
	testConfig.ID = currentConfig.ID
	testConfig.Parent = currentConfig.Parent
	if currentConfig != testConfig {
		t.Errorf("Initial test config is not correct, exp %+v, got %+v", testConfig, currentConfig)
	}

	// wait to make sure we don't create duplicate clients on each scan
	select {
	case testClient = <-newClient:
		t.Fatal("duplicate client created")
	case <-time.After(time.Second * 10):
		// all is well
	}

	// test deleting client
	err = client.SendEdgePoint(nc, currentConfig.ID, currentConfig.Parent,
		data.Point{Type: data.PointTypeTombstone, Value: 1}, true)

	if err != nil {
		t.Errorf("Error sending edge point: %v", err)
	}

	select {
	case <-testClient.stopped:
		// all is well
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting for client to be removed")
	}

	m.Stop(nil)

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
	nc, _, stop, err := test.Server()
	_ = nc

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()
}
