package data

import (
	"testing"
)

func TestNotificationPointRoundTrip(t *testing.T) {
	n := Notification{
		ID:         "9d3c8b6a-1111-2222-3333-444455556666",
		SourceNode: "node-abc",
		Subject:    "temp high",
		Message:    "temp rule fired at boiler",
	}

	p, err := n.Point()
	if err != nil {
		t.Fatal("Error encoding notification:", err)
	}

	if p.Type != PointTypeNotification {
		t.Error("wrong point type:", p.Type)
	}

	if p.DataType != PointDataTypeJSON {
		t.Error("wrong point data type:", p.DataType)
	}

	n2, err := PointToNotification(p)
	if err != nil {
		t.Fatal("Error decoding notification:", err)
	}

	if n2 != n {
		t.Errorf("round trip mismatch: got %+v, expected %+v", n2, n)
	}
}

func TestNotificationPointWrongType(t *testing.T) {
	p := Point{Type: PointTypeValue}
	_, err := PointToNotification(p)
	if err == nil {
		t.Error("expected error decoding notification from non-notification point")
	}
}

func TestMessagePointRoundTrip(t *testing.T) {
	m := Message{
		NotificationID: "9d3c8b6a-1111-2222-3333-444455556666",
		UserID:         "user-abc",
		Phone:          "+15555555555",
		Email:          "joe@example.com",
		Subject:        "temp high",
		Message:        "temp rule fired at boiler",
	}

	p, err := m.Point()
	if err != nil {
		t.Fatal("Error encoding message:", err)
	}

	if p.Type != PointTypeMessage {
		t.Error("wrong point type:", p.Type)
	}

	if p.DataType != PointDataTypeJSON {
		t.Error("wrong point data type:", p.DataType)
	}

	m2, err := PointToMessage(p)
	if err != nil {
		t.Fatal("Error decoding message:", err)
	}

	if m2 != m {
		t.Errorf("round trip mismatch: got %+v, expected %+v", m2, m)
	}
}

func TestMessagePointWrongType(t *testing.T) {
	p := Point{Type: PointTypeValue}
	_, err := PointToMessage(p)
	if err == nil {
		t.Error("expected error decoding message from non-message point")
	}
}
