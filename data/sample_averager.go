package data

import (
	"time"
)

// SampleAverager accumulates samples, and averages them. The average can
// be reset.
type SampleAverager struct {
	total      float64
	count      int
	min        float64
	max        float64
	sampleType string
	sampleTime time.Time
}

// NewSampleAverager initializes and returns an averager
func NewSampleAverager(sampleType string) *SampleAverager {
	return &SampleAverager{
		sampleType: sampleType,
	}
}

// AddSample takes a sample, and adds it to the total
func (sa *SampleAverager) AddSample(s Sample) {
	// avg sample timestamp is set to last sample time
	if s.Time.After(sa.sampleTime) {
		sa.sampleTime = s.Time
	}

	// update statistical values.
	sa.total += s.Value
	sa.count++
	// min
	if sa.min == 0 {
		sa.min = s.Min
	} else if s.Min < sa.min {
		sa.min = s.Min
	}
	// max
	if s.Max > sa.max {
		sa.max = s.Max
	}
}

// ResetAverage sets the accumulated total to zero
func (sa *SampleAverager) ResetAverage() {
	sa.total = 0
	sa.count = 0
	sa.min = 0
	sa.max = 0
}

// GetAverage returns the average of the accumulated samples
func (sa *SampleAverager) GetAverage() Sample {
	return Sample{
		Type:  sa.sampleType,
		Time:  sa.sampleTime,
		Value: sa.total / float64(sa.count),
		Min:   sa.min,
		Max:   sa.max,
	}
}
