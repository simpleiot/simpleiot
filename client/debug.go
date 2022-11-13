package client

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nats-io/nats.go"
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

	switch chunks[0] {
	case "p":
		nodeID := chunks[1]

		// Fetch node so we can print description
		node, err := GetNodes(nc, "none", nodeID, "", false)

		if err != nil {
			return "", fmt.Errorf("Error getting node over nats: %w", err)
		}

		description := node[0].Desc()
		ret += fmt.Sprintf("NODE: %v (%v) (%v)\n", description, node[0].Type, node[0].ID)
		pointLabel := "POINT"
		if len(chunks) == 3 {
			parent := chunks[2]
			ret += fmt.Sprintf("  Parent: %v\n", parent)
			pointLabel = "EDGE POINT"
		}
		_, points, err := DecodeNodePointsMsg(msg)
		if err != nil {
			return "", err
		}

		for _, p := range points {
			ret += fmt.Sprintf("   - %v: %v\n", pointLabel, p)
		}
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
