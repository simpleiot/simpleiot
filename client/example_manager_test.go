package client_test

import (
	"log"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
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
	config        exNode
	stop          chan struct{}
	stopped       chan struct{}
	newPoints     chan client.NewPoints
	newEdgePoints chan client.NewPoints
	chGetConfig   chan chan exNode
}

// newExNodeClient is passed to the NewManager() function call -- when
// a new node is detected, the Manager will call this function to construct
// a new client.
func newExNodeClient(nc *nats.Conn, config exNode) client.Client {
	return &exNodeClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan client.NewPoints),
		newEdgePoints: make(chan client.NewPoints),
	}
}

// Run the main logic for this client and blocks until stopped
func (tnc *exNodeClient) Run() error {
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
			log.Printf("New config: %+v\n", tnc.config)
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

// Stop sends a signal to the Run function to exit
func (tnc *exNodeClient) Stop(_ error) {
	close(tnc.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (tnc *exNodeClient) Points(id string, points []data.Point) {
	tnc.newPoints <- client.NewPoints{id, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (tnc *exNodeClient) EdgePoints(id, parent string, points []data.Point) {
	tnc.newEdgePoints <- client.NewPoints{id, parent, points}
}

func ExampleNewManager() {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		log.Println("Error starting test server: ", err)
	}

	defer stop()

	testConfig := testNode{"", "", "fancy test node", 8118, "admin"}

	// Convert our custom struct to a data.NodeEdge struct
	ne, err := data.Encode(testConfig)
	if err != nil {
		log.Println("Error encoding node: ", err)
	}

	ne.Parent = root.ID

	// hydrate database with test node
	err = client.SendNode(nc, ne, "test")

	if err != nil {
		log.Println("Error sending node: ", err)
	}

	// Create a new manager for nodes of type "testNode". The manager looks for new nodes under the
	// root and if it finds any, it instantiates a new client, and sends point updates to it
	m := client.NewManager(nc, newExNodeClient, nil)
	err = m.Run()

	if err != nil {
		log.Println("Error running: ", err)
	}

	// Now any updates to the node will trigger Points/EdgePoints callbacks in the above client
}
