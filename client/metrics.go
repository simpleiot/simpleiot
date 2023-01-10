package client

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Metrics represents the config of a metrics node type
type Metrics struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Type        string `point:"type"`
	Period      int    `point:"period"`
}

// MetricsClient is a SIOT client used to collect system or app metrics
type MetricsClient struct {
	nc            *nats.Conn
	config        Metrics
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
	lastSend      time.Time
}

// NewMetricsClient ...
func NewMetricsClient(nc *nats.Conn, config Metrics) Client {
	return &MetricsClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
	}
}

// Run the main logic for this client and blocks until stopped
func (m *MetricsClient) Run() error {

	checkPeriod := func() {
		if m.config.Period < 1 {
			m.config.Period = 120
			points := data.Points{
				{Type: data.PointTypePeriod, Value: float64(m.config.Period)},
			}

			err := SendPoints(m.nc, SubjectNodePoints(m.config.ID), points, false)
			if err != nil {
				log.Println("Error sending metrics period: ", err)
			}
		}
	}

	checkPeriod()

done:
	for {
		select {
		case <-m.stop:
			break done

		case pts := <-m.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &m.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypePeriod:
					checkPeriod()
				}
			}

		case pts := <-m.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &m.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

		}
	}

	return nil
}

// Stop sends a signal to the Run function to exit
func (m *MetricsClient) Stop(err error) {
	close(m.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (m *MetricsClient) Points(nodeID string, points []data.Point) {
	m.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (m *MetricsClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	m.newEdgePoints <- NewPoints{nodeID, parentID, points}
}
