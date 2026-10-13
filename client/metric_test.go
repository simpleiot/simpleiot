package client

import (
	"testing"
	"time"
)

// A disabled metric discards samples without sending, so a nil connection
// is never used.
func TestMetricDisabled(t *testing.T) {
	m := NewMetric(nil, "node", "metricTest", 0)
	m.Configure(false, 0)
	m.lastReport = time.Now().Add(-time.Hour)

	for range 3 {
		if err := m.AddSample(1); err != nil {
			t.Fatal("disabled metric returned an error:", err)
		}
	}
}
