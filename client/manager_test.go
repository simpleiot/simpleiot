package client_test

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
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
	newPoints     chan client.NewPoints
	newEdgePoints chan client.NewPoints
	chGetConfig   chan chan testNode
}

func newTestNodeClient(nc *nats.Conn, config testNode) *testNodeClient {
	return &testNodeClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		stopped:       make(chan struct{}),
		newPoints:     make(chan client.NewPoints),
		newEdgePoints: make(chan client.NewPoints),
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
			err := data.MergePoints(pts.ID, pts.Points, &tnc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
		case pts := <-tnc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &tnc.config)
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

func (tnc *testNodeClient) Points(nodeID string, points []data.Point) {
	tnc.newPoints <- client.NewPoints{nodeID, "", points}
}

func (tnc *testNodeClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	tnc.newEdgePoints <- client.NewPoints{nodeID, parentID, points}
}

func (tnc *testNodeClient) getConfig() testNode {
	result := make(chan testNode)
	tnc.chGetConfig <- result
	return <-result
}

func TestManager(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	testConfig := testNode{"ID-testNode", root.ID, "fancy test node", 8080, ""}

	// hydrate database with test data
	err = client.SendNodeType(nc, testConfig, "test")
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
	m := client.NewManager(nc, newTestNodeClientWrapper)

	managerStopped := make(chan struct{})

	startErr := make(chan error)

	go func() {
		err := m.Start()
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
	if currentConfig != testConfig {
		t.Errorf("Initial test config is not correct, exp %+v, got %+v", testConfig, currentConfig)
	}

	// Test point updates
	modifiedDescription := "updated description"

	err = client.SendNodePoint(nc, currentConfig.ID,
		data.Point{Type: "description", Text: modifiedDescription, Origin: "test"}, true)

	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if testClient.getConfig().Description != modifiedDescription {
		t.Error("Description not modified")
	}

	// Test edge point updates to node
	modifiedRole := "user"

	err = client.SendEdgePoint(nc, currentConfig.ID, currentConfig.Parent,
		data.Point{Type: data.PointTypeRole, Text: modifiedRole, Origin: "test"}, true)

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
	nc, root, stop, err := server.TestServer()

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
	m := client.NewManager(nc, newTestNodeClientWrapper)

	managerStopped := make(chan struct{})

	startErr := make(chan error)

	go func() {
		err := m.Start()
		if err != nil {
			startErr <- fmt.Errorf("manager start returned error: %v", err)
		}

		close(managerStopped)
	}()

	// populate with new testNode
	testConfig := testNode{"ID-testnode", root.ID, "fancy test node", 8080, "admin"}
	// populate database with test node
	err = client.SendNodeType(nc, testConfig, "test")
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

	/* FIXME is there a better way to test this??
	// wait to make sure we don't create duplicate clients on each scan
	select {
	case testClient = <-newClient:
		t.Fatal("duplicate client created")
	case <-time.After(time.Second * 10):
		// all is well
	}
	*/

	// test deleting client
	err = client.SendEdgePoint(nc, currentConfig.ID, currentConfig.Parent,
		data.Point{Type: data.PointTypeTombstone, Value: 1, Origin: "test"}, true)

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
	case <-time.After(time.Second * 10):
		t.Fatal("manager did not stop")
	}

	select {
	case <-startErr:
		t.Fatal("Manager start returned an error: ", err)
	default:
		// all is well
	}
}

type testX struct {
	ID          string  `node:"id"`
	Parent      string  `node:"parent"`
	Description string  `point:"description"`
	Role        string  `edgepoint:"role"`
	TestYs      []testY `child:"testY"`
}

type testY struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Role        string `edgepoint:"role"`
}

type testXClient struct {
	nc            *nats.Conn
	config        testX
	stop          chan struct{}
	stopped       chan struct{}
	newPoints     chan client.NewPoints
	newEdgePoints chan client.NewPoints
	chGetConfig   chan chan testX
}

func newTestXClient(nc *nats.Conn, config testX) *testXClient {
	return &testXClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		stopped:       make(chan struct{}),
		newPoints:     make(chan client.NewPoints),
		newEdgePoints: make(chan client.NewPoints),
		chGetConfig:   make(chan chan testX),
	}
}

func (tnc *testXClient) Start() error {
	for {
		select {
		case <-tnc.stop:
			close(tnc.stopped)
			return nil
		case pts := <-tnc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &tnc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
		case pts := <-tnc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &tnc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
		case ch := <-tnc.chGetConfig:
			ch <- tnc.config
		}
	}
}

func (tnc *testXClient) Stop(err error) {
	close(tnc.stop)
}

func (tnc *testXClient) Points(nodeID string, points []data.Point) {
	tnc.newPoints <- client.NewPoints{nodeID, "", points}
}

func (tnc *testXClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	tnc.newEdgePoints <- client.NewPoints{nodeID, parentID, points}
}

func (tnc *testXClient) getConfig() testX {
	result := make(chan testX)
	tnc.chGetConfig <- result
	return <-result
}

func TestManagerChildren(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	// hydrate database with test data
	testXConfig := testX{"ID-X", root.ID, "testX node", "", nil}

	err = client.SendNodeType(nc, testXConfig, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// create child node
	testYConfig := testY{"ID-Y", testXConfig.ID, "testY node", ""}

	err = client.SendNodeType(nc, testYConfig, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	var testClient *testXClient
	newClient := make(chan *testXClient)

	// wrap newTestNodeClient so we can stash a link to testClient
	var newTestXClientWrapper = func(nc *nats.Conn, config testX) client.Client {
		testClient := newTestXClient(nc, config)
		newClient <- testClient
		return testClient
	}

	// Create a new manager for nodes of type "testNode". The manager looks for new nodes under the
	// root and if it finds any, it instantiates a new client, and sends point updates to it
	m := client.NewManager(nc, newTestXClientWrapper)

	managerStopped := make(chan struct{})

	startErr := make(chan error)

	go func() {
		err := m.Start()
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

	_ = currentConfig
	_ = testClient

	if currentConfig.ID != testXConfig.ID {
		t.Fatal("X ID not correct: ", currentConfig.ID)
	}

	if len(currentConfig.TestYs) < 1 {
		t.Fatal("No TestYs")
	}

	if currentConfig.TestYs[0].ID != testYConfig.ID {
		t.Fatal("Y ID not correct")
	}

	if currentConfig.TestYs[0].Description != testYConfig.Description {
		t.Fatal("Y description not correct")
	}

	// Test child point updates
	modifiedDescription := "updated description"

	err = client.SendNodePoint(nc, testYConfig.ID,
		data.Point{Type: "description", Text: modifiedDescription, Origin: "test"}, true)

	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	if testClient.getConfig().TestYs[0].Description != modifiedDescription {
		t.Error("Child Description not modified")
	}

	// Test parent edge point updates
	modifiedRole := "admin"

	err = client.SendEdgePoint(nc, testXConfig.ID, testXConfig.Parent,
		data.Point{Type: data.PointTypeRole, Text: modifiedRole, Origin: "test"}, true)

	if err != nil {
		t.Errorf("Error sending edge point: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if testClient.getConfig().Role != modifiedRole {
		t.Error("Parent Role not modified")
	}

	// Test child edge point updates
	modifiedRole = "user"

	err = client.SendEdgePoint(nc, testYConfig.ID, testYConfig.Parent,
		data.Point{Type: data.PointTypeRole, Text: modifiedRole, Origin: "test"}, true)

	if err != nil {
		t.Errorf("Error sending edge point: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if testClient.getConfig().TestYs[0].Role != modifiedRole {
		t.Error("Child Role not modified")
	}

	// create 2nd child node
	testYConfig2 := testY{"ID-Y2", testXConfig.ID, "testY node 2", ""}

	err = client.SendNodeType(nc, testYConfig2, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// wait for client to be re-created
	select {
	case testClient = <-newClient:
	case <-time.After(time.Second):
		t.Fatal("Test client not re-created")
	}

	if len(testClient.getConfig().TestYs) < 2 {
		t.Fatal("Not seeing new child")
	}

	// remove child node
	err = client.SendEdgePoint(nc, testYConfig2.ID, testYConfig2.Parent,
		data.Point{Type: data.PointTypeTombstone, Value: 1, Origin: "test"}, true)

	if err != nil {
		t.Errorf("Error sending edge point: %v", err)
	}

	// wait for client to be re-created
	select {
	case testClient = <-newClient:
	case <-time.After(time.Second):
		t.Fatal("Test client not re-created")
	}

	if len(testClient.getConfig().TestYs) != 1 {
		t.Fatal("failed to remove child node")
	}
}
