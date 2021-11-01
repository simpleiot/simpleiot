package data

import (
	"time"
)

// PointAverager accumulates points, and averages them. The average can
// be reset.
type PointAverager struct {
	total     float64
	count     int
	min       float64
	max       float64
	pointType string
	pointTime time.Time
}

// NewPointAverager initializes and returns an averager
func NewPointAverager(pointType string) *PointAverager {
	return &PointAverager{
		pointType: pointType,
	}
}

// AddPoint takes a point, and adds it to the total
func (pa *PointAverager) AddPoint(s Point) {
	// avg point timestamp is set to last point time
	if s.Time.After(pa.pointTime) {
		pa.pointTime = s.Time
	}

	// update statistical values.
	pa.total += s.Value
	pa.count++
}

// ResetAverage sets the accumulated total to zero
func (pa *PointAverager) ResetAverage() {
	pa.total = 0
	pa.count = 0
	pa.min = 0
	pa.max = 0
}

// GetAverage returns the average of the accumulated points
func (pa *PointAverager) GetAverage() Point {
	var value float64
	if pa.count == 0 {
		value = 0
	} else {
		value = pa.total / float64(pa.count)
	}

	return Point{
		Type:  pa.pointType,
		Time:  pa.pointTime,
		Value: value,
	}
}
