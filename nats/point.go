package nats

import (
	"fmt"
	"log"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// SendPoint sends a point using the nats protocol
func SendPoint(nc *natsgo.Conn, nodeID string, point *data.Point, ack bool) error {
	subject := fmt.Sprintf("node.%v.points", nodeID)

	points := data.Points{}

	points = append(points, *point)

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
