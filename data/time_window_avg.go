package data

import (
	"time"
)

// TimeWindowAverager accumulates points, and averages them on a fixed time
// period and outputs the average/min/max, etc as a point
type TimeWindowAverager struct {
	start     time.Time
	windowLen time.Duration
	total     float64
	count     int
	callBack  func(Point)
	pointType string
	pointTime time.Time
}

// NewTimeWindowAverager initializes and returns an averager
func NewTimeWindowAverager(windowLen time.Duration, callBack func(Point), pointType string) *TimeWindowAverager {
	return &TimeWindowAverager{
		windowLen: windowLen,
		callBack:  callBack,
		pointType: pointType,
	}
}

// NewPoint takes a point, and if the time window expired, it calls
// the callback function with the a new point which is avg of
// all points since start time.
func (twa *TimeWindowAverager) NewPoint(s Point) {
	// avg point timestamp is set to last point time
	if s.Time.After(twa.pointTime) {
		twa.pointTime = s.Time
	}

	// update statistical values.
	twa.total += s.Value
	twa.count++

	// if time has expired, callback() with avg point
	if time.Since(twa.start) >= twa.windowLen {
		avgPoint := Point{
			Type:  twa.pointType,
			Time:  twa.pointTime,
			Value: twa.total / float64(twa.count),
		}

		twa.callBack(avgPoint)

		// reset statistical values and timestamp
		twa.total = 0
		twa.count = 0
		twa.start = time.Now()
	}
}
