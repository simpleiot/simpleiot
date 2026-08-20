package data

import (
	"encoding/json"
	"fmt"
	"time"
)

// Notification describes something that happened that users may want to
// know about. It travels as a JSON payload in a point of type
// PointTypeNotification on the node that raised it, so it is persisted,
// synchronized between instances, and visible to clients like any other
// point. The point uses a fixed (empty) key, so a node carries only its
// most recent notification -- history lives in the JetStream stream.
type Notification struct {
	// ID is a UUID assigned when the notification is raised. It is carried
	// through to messages and used to deduplicate delivery.
	ID string `json:"id"`
	// SourceNode is the node that triggered the notification (for a rule,
	// the node that satisfied the condition).
	SourceNode string `json:"sourceNode"`
	Subject    string `json:"subject"`
	Message    string `json:"message"`
}

// Point encodes the notification as a point ready to send to a node.
func (n Notification) Point() (Point, error) {
	d, err := json.Marshal(n)
	if err != nil {
		return Point{}, err
	}

	return Point{
		Type:     PointTypeNotification,
		Time:     time.Now(),
		DataType: PointDataTypeJSON,
		Data:     d,
	}, nil
}

// PointToNotification decodes a notification from a point.
func PointToNotification(p Point) (Notification, error) {
	var ret Notification

	if p.Type != PointTypeNotification {
		return ret, fmt.Errorf("point type %v is not a notification", p.Type)
	}

	err := json.Unmarshal(p.Data, &ret)
	return ret, err
}
