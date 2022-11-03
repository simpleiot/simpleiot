package client

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Dump converts displays a NATS message
func Dump(nc *nats.Conn, msg *nats.Msg) error {
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
func String(nc *nats.Conn, msg *nats.Msg) (string, error) {
	ret := ""

	chunks := strings.Split(msg.Subject, ".")

	if len(chunks) < 2 {
		return "", fmt.Errorf("don't know how to decode this subject: %v", msg.Subject)
	}

	if len(chunks) == 2 {
		nodeID := chunks[1]
		// Fetch node so we can print description
		node, err := GetNodes(nc, "", nodeID, "", false)

		if err != nil {
			return "", fmt.Errorf("Error getting node over nats: %w", err)
		}

		description := node[0].Desc()
		ret += fmt.Sprintf("get NODE: %v (%v) (%v)\n", description, node[0].Type, node[0].ID)
	} else if len(chunks) == 3 {
		switch chunks[0] {
		case "node":
			nodeID := chunks[1]

			// Fetch node so we can print description
			node, err := GetNodes(nc, "none", nodeID, "", false)

			if err != nil {
				return "", fmt.Errorf("Error getting node over nats: %w", err)
			}

			description := node[0].Desc()

			ret += fmt.Sprintf("NODE: %v (%v) (%v)\n", description, node[0].Type, node[0].ID)

			switch chunks[2] {
			case "points":
				_, points, err := DecodeNodePointsMsg(msg)
				if err != nil {
					return "", err
				}

				for _, p := range points {
					ret += fmt.Sprintf("   - POINT: %v\n", p)
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
			case "children":
				ret += "   get children\n"
			default:
				log.Println("unknown node op: ", chunks[2])
			}
		case "edge":
			edgeID := chunks[1]
			ret += fmt.Sprintf("EDGE: %v\n", edgeID)

			switch chunks[2] {
			case "points":
				_, points, err := DecodeNodePointsMsg(msg)
				if err != nil {
					return "", err
				}

				for _, p := range points {
					ret += fmt.Sprintf("   - POINT: %v\n", p)
				}
			default:
				log.Println("unknown edge op: ", chunks[2])
			}

		default:
			log.Println("unkown message type: ", chunks[0])
		}
	} else if len(chunks) == 4 {
		if chunks[0] != "node" {
			return "", fmt.Errorf("invalid message, does not start with node: %v", msg.Subject)
		}

		if chunks[3] != "points" {
			return "", fmt.Errorf("invalid message, does not end with points: %v", msg.Subject)
		}

		nodeID := chunks[1]
		parentID := chunks[2]

		node, err := GetNodes(nc, parentID, nodeID, "", false)
		if err != nil {
			return "", fmt.Errorf("Error getting node over nats: %w", err)
		}

		parent, err := GetNodes(nc, "none", parentID, "", false)
		if err != nil {
			return "", fmt.Errorf("Error getting parent over nats: %w", err)
		}

		ret += fmt.Sprintf("EDGE: %v (%v):%v (%v)\n", parent[0].Desc(), parent[0].ID, node[0].Desc(), node[0].ID)

		_, points, err := DecodeNodePointsMsg(msg)
		if err != nil {
			return "", err
		}

		for _, p := range points {
			ret += fmt.Sprintf("   - POINT: %v\n", p)
		}
	} else {
		return "", fmt.Errorf("invalid # of message segments: %v", msg.Subject)
	}

	return ret, nil
}

// Log all nats messages. This function does not block and does not clean up
// after itself.
func Log(natsServer, authToken string) {
	log.Println("Logging all NATS messages")

	opts := EdgeOptions{
		URI:       natsServer,
		AuthToken: authToken,
		NoEcho:    true,
		Disconnected: func() {
			log.Println("NATS Disconnected")
		},
		Reconnected: func() {
			log.Println("NATS Reconnected")
		},
		Closed: func() {
			log.Println("NATS Closed")
			os.Exit(0)
		},
	}

	nc, err := EdgeConnect(opts)

	if err != nil {
		log.Println("Error connecting to NATS server: ", err)
		os.Exit(-1)
	}

	_, err = nc.Subscribe("p.*", func(msg *nats.Msg) {
		err := Dump(nc, msg)
		if err != nil {
			log.Println("Error dumping nats msg: ", err)
		}
	})

	_, err = nc.Subscribe("node.*.not", func(msg *nats.Msg) {
		err := Dump(nc, msg)
		if err != nil {
			log.Println("Error dumping nats msg: ", err)
		}
	})

	_, err = nc.Subscribe("node.*.msg", func(msg *nats.Msg) {
		err := Dump(nc, msg)
		if err != nil {
			log.Println("Error dumping nats msg: ", err)
		}
	})

	_, err = nc.Subscribe("p.*.*", func(msg *nats.Msg) {
		err := Dump(nc, msg)
		if err != nil {
			log.Println("Error dumping nats msg: ", err)
		}
	})

	if err != nil {
		log.Println("Nats subscribe error: ", err)
		os.Exit(-1)
	}

	_, err = nc.Subscribe("node.*", func(msg *nats.Msg) {
		err := Dump(nc, msg)
		if err != nil {
			log.Println("Error dumping nats msg: ", err)
		}
	})

	if err != nil {
		log.Println("Nats subscribe error: ", err)
		os.Exit(-1)
	}

	_, err = nc.Subscribe("edge.*.*", func(msg *nats.Msg) {
		err := Dump(nc, msg)
		if err != nil {
			log.Println("Error dumping nats msg: ", err)
		}
	})

	if err != nil {
		log.Println("Nats subscribe error: ", err)
		os.Exit(-1)
	}
}
