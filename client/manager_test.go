package client_test

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/google/uuid"
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

func (tnc *testNodeClient) Points(nodeID string, points []data.Point) {
	tnc.newPoints <- points
}

func (tnc *testNodeClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	tnc.newEdgePoints <- points
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

	testConfig := testNode{uuid.New().String(), root.ID, "fancy test node", 8080, "admin"}

	ne, err := data.Encode(testConfig)
	if err != nil {
		t.Fatal("Error encoding node: ", err)
	}

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
	testConfig := testNode{uuid.New().String(), "", "fancy test node", 8080, "admin"}

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

type testX struct {
	ID          string  `node:"id"`
	Parent      string  `node:"parent"`
	Description string  `point:"description"`
	TestYs      []testY `child:"testY"`
}

type testY struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
}

type newPoints struct {
	id     string
	points data.Points
}

type testXClient struct {
	nc            *nats.Conn
	config        testX
	stop          chan struct{}
	stopped       chan struct{}
	newPoints     chan newPoints
	newEdgePoints chan newPoints
	chGetConfig   chan chan testX
}

func newTestXClient(nc *nats.Conn, config testX) *testXClient {
	return &testXClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		stopped:       make(chan struct{}),
		newPoints:     make(chan newPoints),
		newEdgePoints: make(chan newPoints),
		chGetConfig:   make(chan chan testX),
	}
}

func (tnc *testXClient) Start() error {
OUTER:
	for {
		select {
		case <-tnc.stop:
			close(tnc.stopped)
			return nil
		case pts := <-tnc.newPoints:
			if pts.id == tnc.config.ID {
				err := data.MergePoints(pts.points, &tnc.config)
				if err != nil {
					log.Println("error merging new points: ", err)
				}
				continue
			}

			for i, y := range tnc.config.TestYs {
				if pts.id == y.ID {
					err := data.MergePoints(pts.points, &tnc.config.TestYs[i])
					if err != nil {
						log.Println("error merging new points: ", err)
					}
					continue OUTER
				}
			}
			log.Println("Error, did not find data structure to merge points in test driver")
		case pts := <-tnc.newEdgePoints:
			err := data.MergeEdgePoints(pts.points, &tnc.config)
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
	tnc.newPoints <- newPoints{nodeID, points}
}

func (tnc *testXClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	tnc.newEdgePoints <- newPoints{nodeID, points}
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

	// hydrate database with test data
	testXConfig := testX{uuid.New().String(), root.ID, "testX node", nil}

	ne, err := data.Encode(testXConfig)
	if err != nil {
		t.Fatal("Error encoding node: ", err)
	}

	err = client.SendNode(nc, ne)

	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// create child node
	testYConfig := testY{uuid.New().String(), testXConfig.ID, "testY node"}
	ne, err = data.Encode(testYConfig)
	if err != nil {
		t.Fatal("Error encoding node: ", err)
	}

	err = client.SendNode(nc, ne)

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
	m := client.NewManager(nc, root.ID, newTestXClientWrapper)

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

	// Test child point updates
	modifiedDescription := "updated description"

	err = client.SendNodePoint(nc, testYConfig.ID,
		data.Point{Type: "description", Text: modifiedDescription}, true)

	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if testClient.getConfig().TestYs[0].Description != modifiedDescription {
		t.Error("Child Description not modified")
	}

	defer stop()
}
