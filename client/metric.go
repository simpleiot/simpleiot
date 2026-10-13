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
	disabled     bool

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

// Configure turns reporting on or off and sets how often it reports. A
// disabled metric discards samples and sends nothing.
func (m *Metric) Configure(enabled bool, reportPeriod time.Duration) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if !enabled && !m.disabled {
		m.avg.ResetAverage()
	}
	if enabled && m.disabled {
		m.lastReport = time.Now()
	}
	m.disabled = !enabled
	m.reportPeriod = reportPeriod
}

// AddSample adds a sample and reports it if reportPeriod has expired
func (m *Metric) AddSample(s float64) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.disabled {
		return nil
	}
	now := time.Now()
	p := data.Point{Time: now}
	p.PutFloat(s)
	m.avg.AddPoint(p)

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
