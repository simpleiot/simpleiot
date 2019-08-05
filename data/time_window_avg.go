package data

import "time"

// TimeWindowAverager accumulates samples, and averages them on a fixed time
// period and outputs the average/min/max, etc as a sample
type TimeWindowAverager struct {
	start     time.Time
	windowLen time.Duration
	total float64
	count int
	callback  func(Sample)
}

func NewTimeWindowAverager(windowLen time.Duration, callback func(Sample)) *TimeWindowAverager {
}

// another option using channle
func (twa *TimeWindowAverager) NewSample(s Sample) {
	// adds sample to slice, and sends data on Out if time window has expired
	if time window experied {
		// calculate average sample (includes min/max)
		twa.callback(avgSample)
	}
}
