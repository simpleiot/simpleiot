package data

import (
	"time"
)

// TimeWindowAverager accumulates samples, and averages them on a fixed time
// period and outputs the average/min/max, etc as a sample
type TimeWindowAverager struct {
	start      time.Time
	windowLen  time.Duration
	total      float64
	count      int
	min        float64
	max        float64
	callBack   func(Sample)
	sampleType string
	sampleTime time.Time
}

// NewTimeWindowAverager initializes and returns an averager
func NewTimeWindowAverager(windowLen time.Duration, callBack func(Sample), sampleType string) *TimeWindowAverager {
	return &TimeWindowAverager{
		windowLen:  windowLen,
		callBack:   callBack,
		sampleType: sampleType,
	}
}

// NewSample takes a sample, and if the time window expired, it calls
// the callback function with the a new sample which is avg of
// all samples since start time.
func (twa *TimeWindowAverager) NewSample(s Sample) {
	// avg sample timestamp is set to last sample time
	if s.Time.After(twa.sampleTime) {
		twa.sampleTime = s.Time
	}

	// update statistical values.
	twa.total += s.Value
	twa.count++
	// min
	if twa.min == 0 {
		twa.min = s.Min
	} else if s.Min < twa.min {
		twa.min = s.Min
	}
	// max
	if s.Max > twa.max {
		twa.max = s.Max
	}

	// if time has expired, callback() with avg sample
	if time.Since(twa.start) >= twa.windowLen {
		avgSample := Sample{
			Type:  twa.sampleType,
			Time:  twa.sampleTime,
			Value: twa.total / float64(twa.count),
			Min:   twa.min,
			Max:   twa.max,
		}

		twa.callBack(avgSample)

		// reset statistical values and timestamp
		twa.total = 0
		twa.count = 0
		twa.min = 0
		twa.max = 0
		twa.start = time.Now()
	}
}
