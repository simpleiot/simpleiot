package client_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// sec22ServerOptions keeps these tests off the ports server.TestServer uses
var sec22ServerOptions = server.Options{
	NatsPort:        8970,
	HTTPPort:        "8971",
	NatsMonitorPort: 8972,
	NatsWSPort:      8973,
	NatsMQTTPort:    8974,
	NatsServer:      "nats://localhost:8970",
	ID:              "sec22",
	DataDir:         filepath.Join(os.TempDir(), "siot-test-sec22"),
}

// TestMsgServiceRecipientFromUserNode verifies that an SMTP service takes
// the recipient from the user node a message point is raised on, so a
// message point that names another address, or one raised on a node that
// is not a user, sends nothing to that address. It also checks that a
// subject cannot add a header to the email.
func TestMsgServiceRecipientFromUserNode(t *testing.T) {
	smtpServer := newFakeSMTPServer(t)
	defer smtpServer.close()

	nc, root, stop, err := server.TestServerOpts(sec22ServerOptions)
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()

	u := client.User{
		ID:     "ID-user",
		Parent: root.ID,
		Email:  "joe@example.com",
	}
	if err := client.SendNodeType(nc, u, "test"); err != nil {
		t.Fatal("Error sending user node:", err)
	}

	svc := client.MsgService{
		ID:          "ID-msgService",
		Parent:      root.ID,
		Description: "test smtp",
		Service:     data.PointValueSMTP,
		URL:         smtpServer.listener.Addr().String(),
		From:        "siot@example.com",
	}
	if err := client.SendNodeType(nc, svc, "test"); err != nil {
		t.Fatal("Error sending msg service node:", err)
	}

	v := client.Variable{
		ID:          "ID-var",
		Parent:      root.ID,
		Description: "test var",
	}
	if err := client.SendNodeType(nc, v, "test"); err != nil {
		t.Fatal("Error sending variable node:", err)
	}

	// wait for the user and msg service clients to start
	time.Sleep(250 * time.Millisecond)

	sendMessage := func(nodeID string, m data.Message) {
		t.Helper()
		p, err := m.Point()
		if err != nil {
			t.Fatal("Error encoding message:", err)
		}
		p.Origin = "test"
		if err := client.SendNodePoint(nc, nodeID, p, true); err != nil {
			t.Fatal("Error sending message point:", err)
		}
	}

	waitEmail := func(timeout time.Duration) (string, bool) {
		select {
		case m := <-smtpServer.msgCh:
			return m, true
		case <-time.After(timeout):
			return "", false
		}
	}

	// a message point on a node that is not a user is ignored
	sendMessage(v.ID, data.Message{
		NotificationID: "ID-not-1",
		Email:          "attacker@example.com",
		Subject:        "from a variable",
		Message:        "should not be sent",
	})
	if m, ok := waitEmail(500 * time.Millisecond); ok {
		t.Fatalf("message point on a variable node was delivered: %v", m)
	}

	// a message point on the user node goes to the user node's address,
	// whatever address the point carries
	sendMessage(u.ID, data.Message{
		NotificationID: "ID-not-2",
		UserID:         u.ID,
		Email:          "attacker@example.com",
		Subject:        "from the user node",
		Message:        "hello",
	})
	email, ok := waitEmail(2 * time.Second)
	if !ok {
		t.Fatal("timeout waiting for SMTP delivery")
	}
	if !strings.Contains(email, "To: joe@example.com") {
		t.Errorf("email not addressed to the user node's email: %v", email)
	}
	if strings.Contains(email, "attacker@example.com") {
		t.Errorf("address from the message point reached the email: %v", email)
	}

	// a subject with CRLF cannot add a header
	sendMessage(u.ID, data.Message{
		NotificationID: "ID-not-3",
		UserID:         u.ID,
		Subject:        "alarm\r\nBcc: attacker@example.com",
		Message:        "hello",
	})
	email, ok = waitEmail(2 * time.Second)
	if !ok {
		t.Fatal("timeout waiting for SMTP delivery")
	}
	for _, line := range strings.Split(email, "\n") {
		if strings.HasPrefix(line, "Bcc:") {
			t.Errorf("subject injected a header: %v", email)
		}
	}
	if !strings.Contains(email, "Subject: alarm  Bcc: attacker@example.com") {
		t.Errorf("subject not folded onto one line: %v", email)
	}
}
