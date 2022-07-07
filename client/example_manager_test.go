package client_test

import (
	"log"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/test"
)

// exNode is decoded data from the client node
type exNode struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Port        int    `point:"port"`
	Role        string `edgepoint:"role"`
}

// exNodeClient contains the logic for this client
type exNodeClient struct {
	nc            *nats.Conn
	config        testNode
	stop          chan struct{}
	stopped       chan struct{}
	newPoints     chan []data.Point
	newEdgePoints chan []data.Point
	chGetConfig   chan chan testNode
}

// newExNodeClient is passed to the NewManager() function call -- when
// a new node is detected, the Manager will call this function to construct
// a new client.
func newExNodeClient(nc *nats.Conn, config testNode) client.Client {
	return &testNodeClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan []data.Point),
		newEdgePoints: make(chan []data.Point),
	}
}

// Start runs the main logic for this client and blocks until stopped
func (tnc *exNodeClient) Start() error {
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
			log.Printf("New config: %+v\n", tnc.config)
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

// Stop sends a signal to the Start function to exit
func (tnc *exNodeClient) Stop(err error) {
	close(tnc.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (tnc *exNodeClient) Points(points []data.Point) {
	tnc.newPoints <- points
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (tnc *exNodeClient) EdgePoints(points []data.Point) {
	tnc.newEdgePoints <- points
}

func ExampleNewManager() {
	nc, root, stop, err := test.Server()

	if err != nil {
		log.Println("Error starting test server: ", err)
	}

	defer stop()

	testConfig := testNode{"", "", "fancy test node", 8080, "admin"}

	// Convert our custom struct to a data.NodeEdge struct
	ne, err := data.Encode(testConfig)
	if err != nil {
		log.Println("Error encoding node: ", err)
	}

	ne.Parent = root.ID

	// hydrate database with test node
	err = client.SendNode(nc, ne)

	if err != nil {
		log.Println("Error sending node: ", err)
	}

	// Create a new manager for nodes of type "testNode". The manager looks for new nodes under the
	// root and if it finds any, it instantiates a new client, and sends point updates to it
	m := client.NewManager(nc, root.ID, newExNodeClient)
	m.Start()

	// Now any updates to the node will tigger Points/EdgePoints callbacks in the above client
}
