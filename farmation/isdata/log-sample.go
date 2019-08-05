package isdata

import "github.com/simpleiot/simpleiot/data"

// Define sample types for the system
const (
	SampleTypePulses            string = "pulses"
	SampleTypeFlowInstantaneous string = "flowInstantaneous"
	SampleTypeFlowWindowAvg     string = "flowWindowAvg"
	SampleTypeAmount            string = "amount"
	SampleTypePressure          string = "pressure"
	SampleTypePressureVRef      string = "pressureVRef"
	SampleTypePressureVSense    string = "pressureVSense"
)

// Samples is used to allow faults to be sorted by timestamp
type Samples []data.Sample

// Below methods allow samples to be automatically sorted by timestamp

// Len return length of slice
func (s Samples) Len() int {
	return len(s)
}

// Swap positions i and j
func (s Samples) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less returns whether timestamp of sample at pos i is after timestamp
// of fault at pos j
func (s Samples) Less(i, j int) bool {
	return s[i].Time.Before(s[j].Time)
}
