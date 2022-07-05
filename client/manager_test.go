package client_test

import (
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
	Description string `point:"description"`
	Port        int    `point:"port"`
}

type testNodeClient struct {
	nc      *nats.Conn
	config  testNode
	stop    chan struct{}
	stopped chan struct{}
}

func newTestNodeClient(nc *nats.Conn, config testNode) *testNodeClient {
	return &testNodeClient{
		nc:      nc,
		config:  config,
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

func (tnc *testNodeClient) Start() error {
	<-tnc.stop
	close(tnc.stopped)
	return nil
}

func (tnc *testNodeClient) Stop(err error) {
	close(tnc.stop)
}

func (tnc *testNodeClient) Update(points []data.Point) {
}

func TestManager(t *testing.T) {
	nc, root, stop, err := test.Server()
	_ = nc

	if err != nil {
		t.Fatal("Error starting test server: ", err)
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

	defer stop()

	ne, err := data.Encode(testNode{"fancy test node", 8080})
	if err != nil {
		t.Fatal("Error encoding node: ", err)
	}

	ne.Parent = root.ID

	// hydrate database with test data
	err = client.SendNode(nc, ne)

	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	m := client.NewManager[testNode](nc, root.ID, newTestNodeClientWrapper)

	managerStopped := make(chan struct{})

	go func() {
		err = m.Start()
		if err != nil {
			t.Fatal("manager start returned error: ", err)
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
