package isdata

import "time"

// FaultType ...
type FaultType int

// define valid fault types
// these faults are stored in a database
// so NEVER replace or remove one --
// only modify by adding to end of list
const (
	FaultTypeIrrOff FaultType = iota
	FaultTypeLowPres
	FaultTypeFlowOffTarget
	FaultShutdownFailed
)

// String returns a message for a fault type
func (ft FaultType) String() string {
	switch ft {
	case FaultTypeIrrOff:
		return "irrigator didnt fill"
	case FaultTypeLowPres:
		return "low pressure"
	case FaultTypeFlowOffTarget:
		return "flow off target"
	case FaultShutdownFailed:
		return "shutdown failed"
	}
	return "unknown fault"
}

// StringVerbose returns a message for the active faults screen
func (ft FaultType) StringVerbose() string {
	switch ft {
	case FaultTypeIrrOff:
		return "Irrigator didnt fill"
	case FaultTypeLowPres:
		return "Shtdwn: low pressure, "
	case FaultTypeFlowOffTarget:
		return "Shtdwn: flow off target, "
	case FaultShutdownFailed:
		return "System failed to shutdown"
	}
	return "unknown fault"
}

// Fault is used to log an event and its time stamp
type Fault struct {
	Fault FaultType
	Time  time.Time
	Value float64 // this is a value that can be the flow or pressure that caused the fault
}

// Faults history for system
type Faults []Fault

// Below methods allow faults to be automatically sorted by time

// Len return length of slice
func (f Faults) Len() int {
	return len(f)
}

// Swap positions i and j
func (f Faults) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// Less returns whether timestamp of fault at pos i is after timestamp
// of fault at pos j
func (f Faults) Less(i, j int) bool {
	return f[i].Time.Before(f[j].Time)
}
