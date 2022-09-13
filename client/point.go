package client

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// SendNodePointCreate sends a node point using the nats protocol and
// creates the node if it does not already exist
func SendNodePointCreate(nc *nats.Conn, nodeID string, point data.Point, ack bool) error {
	return SendNodePointsCreate(nc, nodeID, []data.Point{point}, ack)
}

// SendNodePointsCreate sends a node point using the nats protocol and
// creates the node if it does not already exist
func SendNodePointsCreate(nc *nats.Conn, nodeID string, points data.Points, ack bool) error {
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
func SendNodePoint(nc *nats.Conn, nodeID string, point data.Point, ack bool) error {
	points := data.Points{point}
	return SendNodePoints(nc, nodeID, points, ack)
}

// SendEdgePoint sends a edge point using the nats protocol
func SendEdgePoint(nc *nats.Conn, nodeID, parentID string, point data.Point, ack bool) error {
	points := data.Points{point}
	return SendEdgePoints(nc, nodeID, parentID, points, ack)
}

// SendNodePoints sends node points using the nats protocol
func SendNodePoints(nc *nats.Conn, nodeID string, points data.Points, ack bool) error {
	return SendPoints(nc, SubjectNodePoints(nodeID), points, ack)
}

// SendEdgePoints sends points using the nats protocol
func SendEdgePoints(nc *nats.Conn, nodeID, parentID string, points data.Points, ack bool) error {
	if parentID == "" {
		parentID = "none"
	}
	return SendPoints(nc, SubjectEdgePoints(nodeID, parentID), points, ack)
}

// SendPoints sends points to specified subject
func SendPoints(nc *nats.Conn, subject string, points data.Points, ack bool) error {
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

// SubscribePoints subscripts to point updates for a node and executes a callback
// when new points arrive. stop() can be called to clean up the subscription
func SubscribePoints(nc *nats.Conn, id string, callback func(points []data.Point)) (stop func(), err error) {
	psub, err := nc.Subscribe(SubjectNodePoints(id), func(msg *nats.Msg) {
		points, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points: ", err)
			return
		}

		callback(points)
	})

	return func() {
		psub.Unsubscribe()
	}, err
}

// SubscribeEdgePoints subscripts to edge point updates for a node and executes a callback
// when new points arrive. stop() can be called to clean up the subscription
func SubscribeEdgePoints(nc *nats.Conn, id, parent string, callback func(points []data.Point)) (stop func(), err error) {
	psub, err := nc.Subscribe(SubjectEdgePoints(id, parent), func(msg *nats.Msg) {
		points, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points: ", err)
			return
		}

		callback(points)
	})

	return func() {
		psub.Unsubscribe()
	}, err
}

// newPoints is used to pass new points through channels in client drivers
type newPoints struct {
	id     string
	parent string
	points data.Points
}

func (np newPoints) String() string {
	ret := fmt.Sprintf("New Points: ID: %v Parent: %v\n", np.id, np.parent)
	ret += np.points.String()
	return ret
}
