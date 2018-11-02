package data

import "time"

// Sample represents a value in time
type Sample struct {
	Value float64   `json:"value"`
	Time  time.Time `json:"time"`
}

// NewSample creates a new sample at current time
func NewSample(value float64) Sample {
	return Sample{
		Value: value,
		Time:  time.Now(),
	}
}
