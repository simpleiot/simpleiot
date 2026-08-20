package msg

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// NtfyDefaultServer is used when no server URL is configured
const NtfyDefaultServer = "https://ntfy.sh"

// Ntfy sends messages to an ntfy topic (https://ntfy.sh or a self-hosted
// server)
type Ntfy struct {
	server string
	topic  string
	token  string
	client *http.Client
}

// NewNtfy creates a new ntfy sender. server may be empty, in which case
// the public ntfy.sh server is used. token is an optional access token
// sent as a bearer token.
func NewNtfy(server, topic, token string) *Ntfy {
	if server == "" {
		server = NtfyDefaultServer
	}

	return &Ntfy{
		server: server,
		topic:  topic,
		token:  token,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Send publishes a message to the ntfy topic. The subject is sent as the
// notification title.
func (n *Ntfy) Send(subject, message string) error {
	if n.topic == "" {
		return fmt.Errorf("ntfy topic is not set")
	}

	url := strings.TrimRight(n.server, "/") + "/" + n.topic

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(message))
	if err != nil {
		return err
	}

	if subject != "" {
		req.Header.Set("Title", subject)
	}

	if n.token != "" {
		req.Header.Set("Authorization", "Bearer "+n.token)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("ntfy returned %v: %v", resp.Status, string(body))
	}

	return nil
}
