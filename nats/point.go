package nats

import (
	"fmt"
	"log"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// SendPoints sends points using the nats protocol
func SendPoints(nc *natsgo.Conn, nodeID string, points data.Points, ack bool) error {
	subject := fmt.Sprintf("node.%v.points", nodeID)

	data, err := points.PbEncode()

	if err != nil {
		return err
	}

	if ack {
		msg, err := nc.Request(subject, data, time.Second)

		if err != nil {
			return err
		}

		if string(msg.Data) != "" {
			log.Println("Request returned error: ", string(msg.Data))
		}

	} else {
		if err := nc.Publish(subject, data); err != nil {
			return err
		}
	}

	return err
}

// SendPoint sends a point using the nats protocol
func SendPoint(nc *natsgo.Conn, nodeID string, point data.Point, ack bool) error {
	points := data.Points{point}
	return SendPoints(nc, nodeID, points, ack)
}
