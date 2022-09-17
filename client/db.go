package client

import (
	"log"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Db represents the configuration for a SIOT DB client
type Db struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	URI         string `point:"uri"`
	Org         string `point:"org"`
	Bucket      string `point:"bucket"`
	AuthToken   string `point:"authToken"`
}

// DbClient is a SIOT database client
type DbClient struct {
	nc            *nats.Conn
	config        Db
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
}

// NewDbClient ...
func NewDbClient(nc *nats.Conn, config Db) Client {
	return &DbClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
	}
}

// Start runs the main logic for this client and blocks until stopped
func (dbc *DbClient) Start() error {
	log.Println("Starting db client: ", dbc.config.Description)

	for {
		select {
		case <-dbc.stop:
			log.Println("Stopping db client: ", dbc.config.Description)
			return nil
		case pts := <-dbc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &dbc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

		case pts := <-dbc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &dbc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
		}
	}
}

// Stop sends a signal to the Start function to exit
func (dbc *DbClient) Stop(err error) {
	close(dbc.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (dbc *DbClient) Points(nodeID string, points []data.Point) {
	dbc.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (dbc *DbClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	dbc.newEdgePoints <- NewPoints{nodeID, parentID, points}
}
