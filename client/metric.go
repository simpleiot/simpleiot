package client

import (
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Metric is a type that can be used to track metrics and periodically report
// them to a node point. Data is queued and averaged and then the average is sent
// out as a point.
type Metric struct {
	// config
	nc           *nats.Conn
	nodeID       string
	reportPeriod time.Duration

	// internal state
	lastReport time.Time
	lock       sync.Mutex
	avg        *data.PointAverager
}

// NewMetric creates a new metric
func NewMetric(nc *nats.Conn, nodeID, pointType string, reportPeriod time.Duration) *Metric {
	return &Metric{
		nc:           nc,
		nodeID:       nodeID,
		reportPeriod: reportPeriod,
		lastReport:   time.Now(),
		avg:          data.NewPointAverager(pointType),
	}
}

// SetNodeID -- this is a bit of a hack to get around some init issues
func (m *Metric) SetNodeID(id string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.nodeID = id
}

// AddSample adds a sample and reports it if reportPeriod has expired
func (m *Metric) AddSample(s float64) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	now := time.Now()
	m.avg.AddPoint(data.Point{
		Time:  now,
		Value: s,
	})

	if now.Sub(m.lastReport) > m.reportPeriod {
		err := SendNodePoint(m.nc, m.nodeID, m.avg.GetAverage(), false)
		if err != nil {
			return err
		}

		m.avg.ResetAverage()
		m.lastReport = now
	}

	return nil
}
