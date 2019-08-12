package data

import "time"

// define common sample types
const (
	SampleTypeStartApp    string = "startApp"
	SampleTypeStartSystem        = "startSystem"
	SampleTypeUpdateOS           = "updateOS"
	SampleTypeUpdateApp          = "updateApp"
)

// Sample represents a value in time and should include data that may be
// graphed.
type Sample struct {
	// Type of sample (voltage, current, key, etc)
	Type string `json:"type,omitempty" boltholdIndex:"Type"`

	// ID of the device that provided the sample
	ID string `json:"id,omitempty"`

	// Average OR
	// Instantaneous analog or digital value of the sample.
	// 0 and 1 are used to represent digital values
	Value float64 `json:"value,omitempty"`

	// statistical values that may be calculated
	Min float64 `json:"min,omitempty"`
	Max float64 `json:"max,omitempty"`

	// Time the sample was taken
	Time time.Time `json:"time,omitempty" boltholdKey:"Time" gob:"-"`

	// Duration over which the sample was taken
	Duration time.Duration `json:"duration,omitempty"`

	// Tags are additional attributes used to describe the sample
	// You might add things like friendly name, etc.
	Tags map[string]string `json:"tags,omitempty"`

	// Attributes are additional numerical values
	Attributes map[string]float64 `json:"attributes,omitempty"`
}

// Bool returns a bool representation of value
func (s *Sample) Bool() bool {
	if s.Value == 0 {
		return false
	}
	return true
}
