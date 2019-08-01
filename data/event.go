package data

import "time"

// EventType describes an event. Custom applications that build on top of Simple IoT
// should custom event types at high number above 10,000 to ensure there is not a collision
// between type IDs. Note, these enums should never change.
type EventType int

// define valid events
const (
	// system booted
	EventTypeBoot EventType = 10
	// app had to be restarted due to crash or something
	EventTypeRestartApp
	EventTypeSystemUpdate
	EventTypeAppUpdate
)

// EventLevel is used to describe the "severity" of the event and can be used to
// quickly filter the type of events
type EventLevel int

// define valid events
const (
	EventLevelFault EventLevel = 3
	EventLevelInfo
	EventLevelDebug
)

// Event describes something that happened and might be displayed to user in a
// a sequential log format.
type Event struct {
	Time    time.Time
	Type    EventType
	Level   EventLevel
	Message string
}
