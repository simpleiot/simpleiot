package data

import "time"

// Sample represents a value in time
type Sample struct {
	// Type of sample (voltage, current, key, etc)
	Type string `json:"type,omitempty"`

	// ID of the device that provided the sample
	ID string `json:"id,omitempty"`

	// Instantaneous analog or digital value of the sample.
	// 0 and 1 are used to represent digital values
	Value float64 `json:"value,omitempty"`

	// statistical values that may be calculated
	Avg float64 `json:"avg,omitempty"`
	Min float64 `json:"min,omitempty"`
	Max float64 `json:"max,omitempty"`

	// Time the sample was taken
	Time time.Time `json:"time,omitempty"`

	// Duration over which the sample was taken
	Duration time.Duration `json:"duration,omitempty"`

	// Tags are additional attributes used to describe the sample
	Tags map[string]string `json:"tags,omitempty"`

	// Attributes are additional values
	Attributes map[string]float64 `json:"attributes,omitempty"`
}

// Bool returns a bool representation of value
func (s *Sample) Bool() bool {
	if s.Value == 0 {
		return false
	}
	return true
}

// NewSample creates a new sample at current time
func NewSample(ID, sampleType string, value float64) Sample {
	return Sample{
		ID:    ID,
		Type:  sampleType,
		Value: value,
		Time:  time.Now(),
	}
}
