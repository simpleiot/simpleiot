package client_test

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

type ntfyRequest struct {
	path  string
	title string
	auth  string
	body  string
}

// TestMsgServiceNtfy verifies that a notification with no user in scope
// reaches an ntfy service, and that the same notification arriving by a
// second path is deduplicated.
func TestMsgServiceNtfy(t *testing.T) {
	reqCh := make(chan ntfyRequest, 10)

	ts := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			reqCh <- ntfyRequest{
				path:  r.URL.Path,
				title: r.Header.Get("Title"),
				auth:  r.Header.Get("Authorization"),
				body:  string(body),
			}
			w.WriteHeader(http.StatusOK)
		}))
	defer ts.Close()

	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()

	svc := client.MsgService{
		ID:          "ID-msgService",
		Parent:      root.ID,
		Description: "test ntfy",
		Service:     data.PointValueNtfy,
		URL:         ts.URL,
		Topic:       "siot-test",
		AuthToken:   "test-token",
	}

	err = client.SendNodeType(nc, svc, "test")
	if err != nil {
		t.Fatal("Error sending msg service node:", err)
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

	v2 := client.Variable{
		ID:          "ID-var2",
		Parent:      root.ID,
		Description: "test var 2",
	}

	err = client.SendNodeType(nc, v2, "test")
	if err != nil {
		t.Fatal("Error sending variable node:", err)
	}

	// wait for the msg service client to start
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
	case r := <-reqCh:
		if r.path != "/siot-test" {
			t.Errorf("wrong ntfy path: %v", r.path)
		}
		if r.title != n.Subject {
			t.Errorf("wrong ntfy title: %v", r.title)
		}
		if r.auth != "Bearer test-token" {
			t.Errorf("wrong ntfy auth header: %v", r.auth)
		}
		if r.body != n.Message {
			t.Errorf("wrong ntfy body: %v", r.body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for ntfy delivery")
	}

	// the same notification arriving by a different path must not be
	// delivered again
	err = sendNotification(nc, v2.ID, n)
	if err != nil {
		t.Fatal("Error sending notification:", err)
	}

	select {
	case r := <-reqCh:
		t.Fatalf("duplicate notification was delivered: %+v", r)
	case <-time.After(500 * time.Millisecond):
		// all is well
	}

	// a different notification must be delivered
	n2 := data.Notification{
		ID:         "ID-not-2",
		SourceNode: v.ID,
		Subject:    "second subject",
		Message:    "second message",
	}

	err = sendNotification(nc, v.ID, n2)
	if err != nil {
		t.Fatal("Error sending notification:", err)
	}

	select {
	case r := <-reqCh:
		if r.body != n2.Message {
			t.Errorf("wrong ntfy body: %v", r.body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for second ntfy delivery")
	}
}

// fakeSMTPServer implements just enough of the SMTP protocol to accept one
// message and capture it
type fakeSMTPServer struct {
	listener net.Listener
	msgCh    chan string
}

func newFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("Error starting fake SMTP server:", err)
	}

	s := &fakeSMTPServer{listener: l, msgCh: make(chan string, 10)}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go s.handle(conn)
		}
	}()

	return s
}

func (s *fakeSMTPServer) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	write := func(line string) {
		_, _ = conn.Write([]byte(line + "\r\n"))
	}

	write("220 fake SMTP server ready")

	r := bufio.NewReader(conn)
	var msg strings.Builder
	inData := false

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if inData {
			if line == "." {
				inData = false
				s.msgCh <- msg.String()
				write("250 OK")
				continue
			}
			msg.WriteString(line)
			msg.WriteByte('\n')
			continue
		}

		cmd := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			write("250 fake")
		case strings.HasPrefix(cmd, "MAIL"), strings.HasPrefix(cmd, "RCPT"):
			write("250 OK")
		case strings.HasPrefix(cmd, "DATA"):
			write("354 go ahead")
			inData = true
		case strings.HasPrefix(cmd, "QUIT"):
			write("221 bye")
			return
		default:
			write("250 OK")
		}
	}
}

func (s *fakeSMTPServer) close() {
	_ = s.listener.Close()
}

// TestMsgServiceSMTP verifies the whole chain: a notification raised on a
// sibling node produces a message point on the user node, which the SMTP
// service delivers to the user's email address.
func TestMsgServiceSMTP(t *testing.T) {
	smtpServer := newFakeSMTPServer(t)
	defer smtpServer.close()

	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()

	u := client.User{
		ID:     "ID-user",
		Parent: root.ID,
		Email:  "joe@example.com",
	}

	err = client.SendNodeType(nc, u, "test")
	if err != nil {
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

	err = client.SendNodeType(nc, svc, "test")
	if err != nil {
		t.Fatal("Error sending msg service node:", err)
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

	// wait for the user and msg service clients to start
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

	// the test server also creates an admin user, which receives its own
	// copy, so filter deliveries down to our test user
	waitJoeEmail := func(timeout time.Duration) (string, bool) {
		deadline := time.After(timeout)
		for {
			select {
			case m := <-smtpServer.msgCh:
				if strings.Contains(m, "To: joe@example.com") {
					return m, true
				}
			case <-deadline:
				return "", false
			}
		}
	}

	email, ok := waitJoeEmail(2 * time.Second)
	if !ok {
		t.Fatal("timeout waiting for SMTP delivery")
	}

	if !strings.Contains(email, "Subject: test subject") {
		t.Errorf("email missing Subject header: %v", email)
	}
	if !strings.Contains(email, "test message") {
		t.Errorf("email missing body: %v", email)
	}

	// a duplicate message (e.g. from a mirrored user node) must not be
	// delivered again
	m := data.Message{
		NotificationID: n.ID,
		UserID:         u.ID,
		Email:          u.Email,
		Subject:        n.Subject,
		Message:        n.Message,
	}

	p, err := m.Point()
	if err != nil {
		t.Fatal("Error encoding message:", err)
	}
	p.Origin = "test"

	err = client.SendNodePoint(nc, u.ID, p, true)
	if err != nil {
		t.Fatal("Error sending message point:", err)
	}

	if dup, ok := waitJoeEmail(500 * time.Millisecond); ok {
		t.Fatalf("duplicate message was delivered: %v", dup)
	}
}
