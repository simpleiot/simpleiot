package client_test

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// watchMessages subscribes to message points on a node and returns a
// channel that receives the decoded messages
func watchMessages(t *testing.T, nc *nats.Conn, nodeID string) (chan data.Message, func()) {
	ch := make(chan data.Message, 10)

	sub, err := nc.Subscribe("p."+nodeID+".message.*", func(natsMsg *nats.Msg) {
		points, err := data.DecodePoints(natsMsg.Data)
		if err != nil {
			t.Log("Error decoding points:", err)
			return
		}

		for _, p := range points {
			m, err := data.PointToMessage(p)
			if err != nil {
				t.Log("Error decoding message:", err)
				continue
			}
			ch <- m
		}
	})

	if err != nil {
		t.Fatal("Error subscribing to message points:", err)
	}

	return ch, func() { _ = sub.Unsubscribe() }
}

// sendNotification publishes a notification point on a node
func sendNotification(nc *nats.Conn, nodeID string, n data.Notification) error {
	p, err := n.Point()
	if err != nil {
		return err
	}
	p.Origin = "test"
	return client.SendNodePoint(nc, nodeID, p, true)
}

// TestUserClientNotification verifies that a notification raised on a
// sibling node under the same parent produces a message point on the user
// node carrying the user's contact information.
func TestUserClientNotification(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()

	u := client.User{
		ID:        "ID-user",
		Parent:    root.ID,
		FirstName: "Joe",
		Phone:     "+15555555555",
		Email:     "joe@example.com",
	}

	err = client.SendNodeType(nc, u, "test")
	if err != nil {
		t.Fatal("Error sending user node:", err)
	}

	v := client.Variable{
		ID:          "ID-var",
		Parent:      root.ID,
		Description: "test var",
	}

	err = client.SendNodeType(nc, v, "test")
	if err != nil {
		t.Fatal("Error sending variable node:", err)
	}

	msgCh, msgStop := watchMessages(t, nc, u.ID)
	defer msgStop()

	// wait for the user client to start
	time.Sleep(250 * time.Millisecond)

	n := data.Notification{
		ID:         "ID-not-1",
		SourceNode: v.ID,
		Subject:    "test subject",
		Message:    "test message",
	}

	err = sendNotification(nc, v.ID, n)
	if err != nil {
		t.Fatal("Error sending notification:", err)
	}

	select {
	case m := <-msgCh:
		if m.NotificationID != n.ID {
			t.Errorf("wrong notification ID: %v", m.NotificationID)
		}
		if m.UserID != u.ID {
			t.Errorf("wrong user ID: %v", m.UserID)
		}
		if m.Phone != u.Phone {
			t.Errorf("wrong phone: %v", m.Phone)
		}
		if m.Email != u.Email {
			t.Errorf("wrong email: %v", m.Email)
		}
		if m.Subject != n.Subject {
			t.Errorf("wrong subject: %v", m.Subject)
		}
		if m.Message != n.Message {
			t.Errorf("wrong message: %v", m.Message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message point on user node")
	}
}

// TestUserClientNotificationOutOfScope verifies that a notification raised
// outside the user's parent subtree does not produce a message.
func TestUserClientNotificationOutOfScope(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()

	g := client.Group{
		ID:          "ID-group",
		Parent:      root.ID,
		Description: "test group",
	}

	err = client.SendNodeType(nc, g, "test")
	if err != nil {
		t.Fatal("Error sending group node:", err)
	}

	u := client.User{
		ID:     "ID-user",
		Parent: g.ID,
		Email:  "joe@example.com",
	}

	err = client.SendNodeType(nc, u, "test")
	if err != nil {
		t.Fatal("Error sending user node:", err)
	}

	// variable node outside the group
	v := client.Variable{
		ID:          "ID-var",
		Parent:      root.ID,
		Description: "test var",
	}

	err = client.SendNodeType(nc, v, "test")
	if err != nil {
		t.Fatal("Error sending variable node:", err)
	}

	msgCh, msgStop := watchMessages(t, nc, u.ID)
	defer msgStop()

	// wait for the user client to start
	time.Sleep(250 * time.Millisecond)

	err = sendNotification(nc, v.ID, data.Notification{
		ID:         "ID-not-1",
		SourceNode: v.ID,
		Subject:    "test subject",
		Message:    "test message",
	})
	if err != nil {
		t.Fatal("Error sending notification:", err)
	}

	select {
	case m := <-msgCh:
		t.Fatalf("unexpected message for out of scope notification: %+v", m)
	case <-time.After(500 * time.Millisecond):
		// all is well
	}
}

// TestUserClientNoContactInfo verifies that a user with no phone or email
// does not emit message points.
func TestUserClientNoContactInfo(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()

	u := client.User{
		ID:        "ID-user",
		Parent:    root.ID,
		FirstName: "Joe",
	}

	err = client.SendNodeType(nc, u, "test")
	if err != nil {
		t.Fatal("Error sending user node:", err)
	}

	v := client.Variable{
		ID:          "ID-var",
		Parent:      root.ID,
		Description: "test var",
	}

	err = client.SendNodeType(nc, v, "test")
	if err != nil {
		t.Fatal("Error sending variable node:", err)
	}

	msgCh, msgStop := watchMessages(t, nc, u.ID)
	defer msgStop()

	time.Sleep(250 * time.Millisecond)

	err = sendNotification(nc, v.ID, data.Notification{
		ID:      "ID-not-1",
		Subject: "test subject",
		Message: "test message",
	})
	if err != nil {
		t.Fatal("Error sending notification:", err)
	}

	select {
	case m := <-msgCh:
		t.Fatalf("unexpected message for user with no contact info: %+v", m)
	case <-time.After(500 * time.Millisecond):
		// all is well
	}
}
