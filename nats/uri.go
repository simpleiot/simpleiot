package nats

import (
	"fmt"
	"strings"
)

// returns protocol, server, port, err
func parseURI(uri string) (string, string, string, error) {
	uri = strings.Trim(uri, " ")
	parts := strings.Split(uri, "://")
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("URI %v does not contain ://", uri)
	}

	proto := parts[0]
	server := parts[1]

	parts = strings.Split(server, ":")

	port := ""
	if len(parts) > 1 {
		server = parts[0]
		port = parts[1]
	}

	return proto, server, port, nil
}

func sanitizeURI(uri string) (string, error) {
	// check if port not specified
	proto, server, port, err := parseURI(uri)
	if err != nil {
		return uri, err
	}

	if port == "" {
		switch proto {
		case "ws":
			port = "80"
		case "wss":
			port = "443"
		default:
			port = "4222"
		}
	}

	return fmt.Sprintf("%v://%v:%v", proto, server, port), nil
}
