package isdata

import "time"

// Fault is used to log an event and its time stamp
type Fault struct {
	Fault string
	Time  time.Time
}

// Faults history for system
type Faults []Fault
