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
)

// String returns a message for a fault type
func (ft FaultType) String() string {
	switch ft {
	case FaultTypeIrrOff:
		return "irrigator didnt fill"
	case FaultTypeLowPres:
		return "min pressure too low: possible leak"
	}
	return "unknown fault"
}

// Fault is used to log an event and its time stamp
type Fault struct {
	Fault FaultType
	Time  time.Time
}

// Faults history for system
type Faults []Fault
