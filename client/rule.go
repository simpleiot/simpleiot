package client

import (
	"log"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Rule represent a rule node config
type Rule struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Disable     bool   `point:"disable"`
}

// RuleClient is a SIOT client used to run rules
type RuleClient struct {
	nc            *nats.Conn
	config        Rule
	stop          chan struct{}
	newPoints     chan []data.Point
	newEdgePoints chan []data.Point
}

// NewRuleClient ...
func NewRuleClient(nc *nats.Conn, config Rule) Client {
	return &RuleClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan []data.Point),
		newEdgePoints: make(chan []data.Point),
	}
}

// Start runs the main logic for this client and blocks until stopped
func (rc *RuleClient) Start() error {
	for {
		select {
		case <-rc.stop:
			return nil
		case pts := <-rc.newPoints:
			err := data.MergePoints(pts, &rc.config)
			if err != nil {
				log.Println("error merging rule points: ", err)
			}
		case pts := <-rc.newEdgePoints:
			err := data.MergeEdgePoints(pts, &rc.config)
			if err != nil {
				log.Println("error merging rule edge points: ", err)
			}
		}
	}
}

// Stop sends a signal to the Start function to exit
func (rc *RuleClient) Stop(err error) {
	close(rc.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (rc *RuleClient) Points(nodeID string, points []data.Point) {
	rc.newPoints <- points
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (rc *RuleClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	rc.newEdgePoints <- points
}
