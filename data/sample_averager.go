package data

import (
	"time"
)

// SampleAverager accumulates samples, and averages them. The average can
// be reset.
type SampleAverager struct {
	Total      float64
	Count      int
	Min        float64
	Max        float64
	SampleType string
	SampleTime time.Time
}

// NewSampleAverager initializes and returns an averager
func NewSampleAverager(sampleType string) *SampleAverager {
	return &SampleAverager{
		SampleType: sampleType,
	}
}

// AddSample takes a sample, and adds it to the total
func (sa *SampleAverager) AddSample(s Sample) {
	// avg sample timestamp is set to last sample time
	if s.Time.After(sa.SampleTime) {
		sa.SampleTime = s.Time
	}

	// update statistical values.
	sa.Total += s.Value
	sa.Count++
	// min
	if sa.Min == 0 {
		sa.Min = s.Min
	} else if s.Min < sa.Min {
		sa.Min = s.Min
	}
	// max
	if s.Max > sa.Max {
		sa.Max = s.Max
	}
}

// ResetAverage sets the accumulated total to zero
func (sa *SampleAverager) ResetAverage() {
	sa.Total = 0
	sa.Count = 0
	sa.Min = 0
	sa.Max = 0
}

// GetAverage returns the average of the accumulated samples
func (sa *SampleAverager) GetAverage() Sample {
	var value float64
	if sa.Count == 0 {
		value = 0
	} else {
		value = sa.Total / float64(sa.Count)
	}

	return Sample{
		Type:  sa.SampleType,
		Time:  sa.SampleTime,
		Value: value,
		Min:   sa.Min,
		Max:   sa.Max,
	}
}
