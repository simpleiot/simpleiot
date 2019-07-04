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
)

// String returns a message for a fault type
func (ft FaultType) String() string {
	switch ft {
	case FaultTypeIrrOff:
		return "IRRIGATOR DIDN'T FILL"
	}
	return "UNKNOWN FAULT"
}

// Fault is used to log an event and its time stamp
type Fault struct {
	Fault FaultType
	Time  time.Time
}

// Faults history for system
type Faults []Fault
