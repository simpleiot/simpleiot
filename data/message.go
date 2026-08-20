package data

import (
	"encoding/json"
	"fmt"
	"time"
)

// Message is a notification addressed to a particular user. It travels as
// a JSON payload in a point of type PointTypeMessage on the user node that
// generated it. Like notifications, the point uses a fixed (empty) key, so
// the user node carries only its most recent message.
type Message struct {
	// NotificationID is carried through unchanged from the notification
	// that generated this message, so a service can recognize two messages
	// generated from one notification by different instances of a mirrored
	// user node.
	NotificationID string `json:"notificationID"`
	UserID         string `json:"userID"`
	Phone          string `json:"phone"`
	Email          string `json:"email"`
	Subject        string `json:"subject"`
	Message        string `json:"message"`
}

// Point encodes the message as a point ready to send to a node.
func (m Message) Point() (Point, error) {
	d, err := json.Marshal(m)
	if err != nil {
		return Point{}, err
	}

	return Point{
		Type:     PointTypeMessage,
		Time:     time.Now(),
		DataType: PointDataTypeJSON,
		Data:     d,
	}, nil
}

// PointToMessage decodes a message from a point.
func PointToMessage(p Point) (Message, error) {
	var ret Message

	if p.Type != PointTypeMessage {
		return ret, fmt.Errorf("point type %v is not a message", p.Type)
	}

	err := json.Unmarshal(p.Data, &ret)
	return ret, err
}
