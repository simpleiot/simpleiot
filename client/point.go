package client

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

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

// SendNodePoints sends node points using the nats protocol.
func SendNodePoints(nc *nats.Conn, nodeID string, points data.Points, ack bool) error {
	return SendPoints(nc, SubjectNodePoints(nodeID), points, ack)
}

// SendEdgePoints sends edge points using the nats protocol.
// Edge points are sent as a batch (single message) because the store requires
// atomicity — new edges need nodeType and tombstone in the same message.
func SendEdgePoints(nc *nats.Conn, nodeID, parentID string, points data.Points, ack bool) error {
	if parentID == "" {
		parentID = "none"
	}
	return SendPointsBatch(nc, SubjectEdgePoints(nodeID, parentID), points, ack)
}

// SendPoints sends points to specified base subject. Each point is sent as a
// separate NATS message with type/key appended to the subject:
// <baseSubject>.<type>.<key>
func SendPoints(nc *nats.Conn, baseSubject string, points data.Points, ack bool) error {
	for _, p := range points {
		if p.Time.IsZero() {
			p.Time = time.Now()
		}

		typ := p.Type
		if typ == "" {
			typ = "_"
		}
		key := p.Key
		if key == "" {
			key = "0"
		}
		subject := baseSubject + "." + typ + "." + key
		pts := data.Points{p}
		d := pts.Encode()

		if ack {
			msg, err := nc.Request(subject, d, time.Second)
			if err != nil {
				return err
			}
			if len(msg.Data) > 0 {
				return errors.New(string(msg.Data))
			}
		} else {
			if err := nc.Publish(subject, d); err != nil {
				return err
			}
		}
	}

	return nil
}

// SendPointsBatch sends all points in a single NATS message. Used for edge
// points where atomicity is required (e.g., nodeType + tombstone must arrive together).
func SendPointsBatch(nc *nats.Conn, subject string, points data.Points, ack bool) error {
	for i := range points {
		if points[i].Time.IsZero() {
			points[i].Time = time.Now()
		}
	}
	d := points.Encode()

	if ack {
		msg, err := nc.Request(subject, d, time.Second)
		if err != nil {
			return err
		}
		if len(msg.Data) > 0 {
			return errors.New(string(msg.Data))
		}
	} else {
		if err := nc.Publish(subject, d); err != nil {
			return err
		}
	}

	return nil
}

// SubscribePoints subscripts to point updates for a node and executes a callback
// when new points arrive. stop() can be called to clean up the subscription
func SubscribePoints(nc *nats.Conn, id string, callback func(points []data.Point)) (stop func(), err error) {
	// Subscribe to p.<id>.> to match p.<id>.<type>.<key>
	psub, err := nc.Subscribe(SubjectNodePoints(id)+".>", func(msg *nats.Msg) {
		points, err := data.DecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points:", err)
			return
		}

		callback(points)
	})

	return func() {
		err := psub.Unsubscribe()
		if err != nil {
			log.Println("Unsubscribe points error:", err)
		}
	}, err
}

// SubscribeEdgePoints subscripts to edge point updates for a node and executes a callback
// when new points arrive. stop() can be called to clean up the subscription
func SubscribeEdgePoints(nc *nats.Conn, id, parent string, callback func(points []data.Point)) (stop func(), err error) {
	psub, err := nc.Subscribe(SubjectEdgePoints(id, parent), func(msg *nats.Msg) {
		points, err := data.DecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points:", err)
			return
		}

		callback(points)
	})

	return func() {
		err := psub.Unsubscribe()
		if err != nil {
			log.Println("Unsubscribe points error:", err)
		}
	}, err
}

// NewPoints is used to pass new points through channels in client drivers
type NewPoints struct {
	ID     string
	Parent string
	Points data.Points
}

func (np NewPoints) String() string {
	ret := fmt.Sprintf("New Points: ID:%v", np.ID)
	if np.Parent != "" {
		ret += fmt.Sprintf("  Parent:%v", np.Parent)
	}
	ret += "\n"
	ret += np.Points.String()
	return ret
}
