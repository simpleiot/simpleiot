package nats

import (
	"errors"
	"fmt"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// SendNodePointCreate sends a node point using the nats protocol and
// creates the node if it does not already exist
func SendNodePointCreate(nc *natsgo.Conn, nodeID string, point data.Point, ack bool) error {
	return SendNodePointsCreate(nc, nodeID, []data.Point{point}, ack)
}

// SendNodePointsCreate sends a node point using the nats protocol and
// creates the node if it does not already exist
func SendNodePointsCreate(nc *natsgo.Conn, nodeID string, points data.Points, ack bool) error {
	_, err := GetNode(nc, nodeID, "none")
	newNode := false
	if err != nil {
		if err != data.ErrDocumentNotFound {
			return fmt.Errorf("GetNode error: %w", err)
		}

		newNode = true
	}

	err = SendNodePoints(nc, nodeID, points, ack)
	if err != nil {
		return fmt.Errorf("SendNodePoints error: %w", err)
	}

	if newNode {
		err := SendEdgePoint(nc, nodeID, "", data.Point{
			Type:  data.PointTypeTombstone,
			Value: 0,
		}, true)

		if err != nil {
			return fmt.Errorf("SendEdgePoint error: %w", err)
		}
	}

	return nil
}

// SendNodePoint sends a node point using the nats protocol
func SendNodePoint(nc *natsgo.Conn, nodeID string, point data.Point, ack bool) error {
	points := data.Points{point}
	return SendNodePoints(nc, nodeID, points, ack)
}

// SendEdgePoint sends a edge point using the nats protocol
func SendEdgePoint(nc *natsgo.Conn, nodeID, parentID string, point data.Point, ack bool) error {
	points := data.Points{point}
	return SendEdgePoints(nc, nodeID, parentID, points, ack)
}

// SendNodePoints sends node points using the nats protocol
func SendNodePoints(nc *natsgo.Conn, nodeID string, points data.Points, ack bool) error {
	return sendPoints(nc, SubjectNodePoints(nodeID), points, ack)
}

// SendEdgePoints sends points using the nats protocol
func SendEdgePoints(nc *natsgo.Conn, nodeID, parentID string, points data.Points, ack bool) error {
	if parentID == "" {
		parentID = "none"
	}
	return sendPoints(nc, SubjectEdgePoints(nodeID, parentID), points, ack)
}

func sendPoints(nc *natsgo.Conn, subject string, points data.Points, ack bool) error {
	for i := range points {
		if points[i].Time.IsZero() {
			points[i].Time = time.Now()
		}
	}
	data, err := points.ToPb()

	if err != nil {
		return err
	}

	if ack {
		msg, err := nc.Request(subject, data, time.Second)

		if err != nil {
			return err
		}

		if len(msg.Data) > 0 {
			return errors.New(string(msg.Data))
		}

	} else {
		if err := nc.Publish(subject, data); err != nil {
			return err
		}
	}

	return err
}
