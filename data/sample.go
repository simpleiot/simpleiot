package data

import "time"

// Sample represents a value in time
type Sample struct {
	ID    string    `json:"id"`
	Value float64   `json:"value"`
	Time  time.Time `json:"time"`
}

// NewSample creates a new sample at current time
func NewSample(ID string, value float64) Sample {
	return Sample{
		ID:    ID,
		Value: value,
		Time:  time.Now(),
	}
}
