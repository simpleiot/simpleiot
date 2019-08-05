package data

import "time"

// TimeWindowAverager accumulates samples, and averages them on a fixed time
// period and outputs the average/min/max, etc as a sample
type TimeWindowAverager struct {
	start     time.Time
	windowLen time.Duration
	total     float64
	count     int
	min       float64
	max       float64
	callback  func(Sample)
}

// NewTimeWindowAverager initializes and returns an averager
func NewTimeWindowAverager(windowLen time.Duration, callback func(Sample)) *TimeWindowAverager {
	return &TimeWindowAverager{
		windowLen: windowLen,
		callback:  callback,
	}
}

// NewSample takes a sample, and if the window time has expired, it calls
// the callback function with the avg, min, max of all samples since start time
func (twa *TimeWindowAverager) NewSample(s Sample) {
	// update statistical values
	twa.total += s.Value
	twa.count++
	if s.Min < twa.min {
		twa.min = s.Min
	}
	if s.Max > twa.max {
		twa.max = s.Max
	}

	// if time has expired, return statistical data with callback function
	if time.Since(twa.start) >= twa.windowLen {
		avgSample := Sample{
			Value: twa.total / float64(twa.count),
			Min:   twa.min,
			Max:   twa.max,
		}

		twa.callback(avgSample)

		// reset statistical values and timestamp
		twa.total = 0
		twa.count = 0
		twa.min = 0
		twa.max = 0
		twa.start = time.Now()
	}
}
