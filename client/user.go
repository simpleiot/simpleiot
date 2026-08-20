package client

import (
	"fmt"
	"log"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// User represents a user node
type User struct {
	ID        string `node:"id"`
	Parent    string `node:"parent"`
	FirstName string `point:"firstName"`
	LastName  string `point:"lastName"`
	Phone     string `point:"phone"`
	Email     string `point:"email"`
	Pass      string `point:"pass"`
}

// UserClient watches for notifications raised anywhere in the parent's
// subtree and addresses a copy to this user by emitting a message point on
// the user's own node, populated with the user's contact information.
// Message service clients (Twilio, SMTP) consume these message points and
// perform the delivery.
type UserClient struct {
	nc               *nats.Conn
	config           User
	stop             chan struct{}
	newPoints        chan NewPoints
	newEdgePoints    chan NewPoints
	newNotifications chan data.Notification
	upSub            *nats.Subscription
}

// NewUserClient returns a new UserClient using its configuration read from
// the Client Manager
func NewUserClient(nc *nats.Conn, config User) Client {
	return &UserClient{
		nc:               nc,
		config:           config,
		stop:             make(chan struct{}),
		newPoints:        make(chan NewPoints),
		newEdgePoints:    make(chan NewPoints),
		newNotifications: make(chan data.Notification),
	}
}

// Run the main logic for this client and blocks until stopped
func (uc *UserClient) Run() error {
	// watch for notifications raised anywhere in the parent's subtree
	subject := fmt.Sprintf("up.%v.>", uc.config.Parent)

	var err error
	uc.upSub, err = uc.nc.Subscribe(subject, func(msg *nats.Msg) {
		points, err := data.DecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points in user upSub:", err)
			return
		}

		// up.<parentId>.<nodeId>.<type>.<key> = 5 chunks for node points
		chunks := strings.Split(msg.Subject, ".")
		if len(chunks) != 5 {
			return
		}

		for _, p := range points {
			if p.Type != data.PointTypeNotification || p.Tombstone%2 != 0 {
				continue
			}

			n, err := data.PointToNotification(p)
			if err != nil {
				log.Println("User client error decoding notification:", err)
				continue
			}

			uc.newNotifications <- n
		}
	})

	if err != nil {
		return fmt.Errorf("User client error subscribing to upsub: %w", err)
	}

done:
	for {
		select {
		case <-uc.stop:
			break done

		case n := <-uc.newNotifications:
			if uc.config.Phone == "" && uc.config.Email == "" {
				// no way to contact this user
				continue
			}

			m := data.Message{
				NotificationID: n.ID,
				UserID:         uc.config.ID,
				Phone:          uc.config.Phone,
				Email:          uc.config.Email,
				Subject:        n.Subject,
				Message:        n.Message,
			}

			p, err := m.Point()
			if err != nil {
				log.Println("User client error encoding message:", err)
				continue
			}

			err = SendNodePoint(uc.nc, uc.config.ID, p, false)
			if err != nil {
				log.Println("User client error sending message point:", err)
			}

		case pts := <-uc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &uc.config)
			if err != nil {
				log.Println("error merging user points:", err)
			}

		case pts := <-uc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &uc.config)
			if err != nil {
				log.Println("error merging user edge points:", err)
			}
		}
	}

	return uc.upSub.Unsubscribe()
}

// Stop sends a signal to the Run function to exit
func (uc *UserClient) Stop(_ error) {
	close(uc.stop)
}

// Points is called by the Manager when new points for this node are received.
func (uc *UserClient) Points(nodeID string, points []data.Point) {
	uc.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this node
// are received.
func (uc *UserClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	uc.newEdgePoints <- NewPoints{nodeID, parentID, points}
}
