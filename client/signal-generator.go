package client

import (
	"log"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// SignalGenerator config
type SignalGenerator struct {
	ID          string  `node:"id"`
	Parent      string  `node:"parent"`
	Description string  `point:"description"`
	Frequency   float64 `point:"frequency"`
	Amplitude   float64 `point:"amplitude"`
	Offset      float64 `point:"offset"`
	SampleRate  float64 `point:"sampleRate"`
	Value       float64 `point:"value"`
	Units       string  `point:"units"`
}

// SignalGeneratorClient for signal generator nodes
type SignalGeneratorClient struct {
	nc            *nats.Conn
	config        SignalGenerator
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
}

// NewSignalGeneratorClient ...
func NewSignalGeneratorClient(nc *nats.Conn, config SignalGenerator) Client {
	return &SignalGeneratorClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
	}
}

// Start runs the main logic for this client and blocks until stopped
func (dbc *SignalGeneratorClient) Start() error {
	log.Println("Starting sig gen client: ", dbc.config.Description)

done:
	for {
		select {
		case <-dbc.stop:
			log.Println("Stopping db client: ", dbc.config.Description)
			break done
		case pts := <-dbc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &dbc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeFrequency, data.PointTypeAmplitude,
					data.PointTypeOffset, data.PointTypeSampleRate:
					// restart generator
				}
			}

		case pts := <-dbc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &dbc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
		}
	}

	// clean up
	return nil
}

// Stop sends a signal to the Start function to exit
func (dbc *SignalGeneratorClient) Stop(err error) {
	close(dbc.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (dbc *SignalGeneratorClient) Points(nodeID string, points []data.Point) {
	dbc.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (dbc *SignalGeneratorClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	dbc.newEdgePoints <- NewPoints{nodeID, parentID, points}
}
