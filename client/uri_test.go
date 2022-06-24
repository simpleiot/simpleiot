package client

import (
	"testing"
)

func TestNatsURIParts(t *testing.T) {
	proto, server, port, err := parseURI("wss://myserver.com:443")
	if err != nil {
		t.Error(err)
	}

	if proto != "wss" {
		t.Error("Wrong proto, expected wss, got: ", proto)
	}

	if server != "myserver.com" {
		t.Error("Wrong server, expected myserver.com, got: ", server)
	}

	if port != "443" {
		t.Error("Wrong port, expected 443, got: ", port)
	}
}

func TestNatsURIPartsNoPort(t *testing.T) {
	proto, server, port, err := parseURI("wss://myserver.com")
	if err != nil {
		t.Error(err)
	}

	if proto != "wss" {
		t.Error("Wrong proto, expected wss, got: ", proto)
	}

	if server != "myserver.com" {
		t.Error("Wrong server, expected myserver.com, got: ", server)
	}

	if port != "" {
		t.Error("Wrong blank port, got: ", port)
	}
}

func TestNatsURIPartsSpaces(t *testing.T) {
	proto, server, port, err := parseURI("  wss://myserver.com:443  ")
	if err != nil {
		t.Error(err)
	}

	if proto != "wss" {
		t.Error("Wrong proto, expected wss, got: ", proto)
	}

	if server != "myserver.com" {
		t.Error("Wrong server, expected myserver.com, got: ", server)
	}

	if port != "443" {
		t.Error("Wrong port, expected 443, got: ", port)
	}
}

type sanitizeTests struct {
	in  string
	exp string
}

func TestSanitizeURI(t *testing.T) {
	tests := []sanitizeTests{
		{"nats://myserver.com", "nats://myserver.com:4222"},
		{"ws://myserver.com:8080", "ws://myserver.com:8080"},
		{"ws://myserver.com", "ws://myserver.com:80"},
		{"wss://myserver.com", "wss://myserver.com:443"},
		{"wsss://myserver.com", "wsss://myserver.com:4222"},
	}

	for _, test := range tests {
		s, err := sanitizeURI(test.in)
		if err != nil {
			t.Errorf("Error sanitizing %v: %v", test.in, err)
		}

		if s != test.exp {
			t.Errorf("Error sanitizing %v, got %v, expected %v", test.in, s, test.exp)
		}
	}
}

func TestSanitizeURIError(t *testing.T) {

	_, err := sanitizeURI("nats:/myserver.com")
	if err == nil {
		t.Error("Expected error")
	}
}
