package nats

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Dump converts displays a NATS message
func Dump(nc *natsgo.Conn, msg *natsgo.Msg) error {
	s, err := String(nc, msg)
	if err != nil {
		return err
	}
	if s != "" {
		log.Printf(s)
	}

	return nil
}

// String converts a NATS message to a string
func String(nc *natsgo.Conn, msg *natsgo.Msg) (string, error) {
	ret := ""

	chunks := strings.Split(msg.Subject, ".")

	nodeID := chunks[1]

	if len(chunks) < 3 {
		return "", fmt.Errorf("don't know how to decode this subject: %v", msg.Subject)
	}

	if chunks[0] != "node" {
		return "", errors.New("can only decode node messages")
	}

	// Fetch node so we can print description
	nodeMsg, err := nc.Request("node."+nodeID, nil, time.Second)

	if err != nil {
		return "", fmt.Errorf("Error getting node over NATS: %w", err)
	}

	node, err := data.PbDecodeNode(nodeMsg.Data)

	if err != nil {
		return "", fmt.Errorf("Error decoding node data from server: %w", err)
	}

	description := node.Desc()

	ret += fmt.Sprintf("NODE: %v (%v) (%v)\n", description, node.Type, node.ID)

	switch chunks[2] {
	case "points":
		_, points, err := DecodeNodePointsMsg(msg)
		if err != nil {
			return "", err
		}

		for _, p := range points {
			if p.Text != "" {
				ret += fmt.Sprintf("   - POINT: %v: %v\n", p.Type, p.Text)
			} else {
				ret += fmt.Sprintf("   - POINT: %v: %v\n", p.Type, p.Value)
			}
		}

	case "not":
		not, err := data.PbDecodeNotification(msg.Data)
		if err != nil {
			return "", err
		}
		ret += fmt.Sprintf("    - Notification: %+v\n", not)
	case "msg":
		message, err := data.PbDecodeMessage(msg.Data)
		if err != nil {
			return "", err
		}
		ret += fmt.Sprintf("    - Message: %+v\n", message)
	}

	return ret, nil
}
