package client

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/msg"
)

// MsgService represents a message service node config (Twilio, SMTP, ntfy)
type MsgService struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	// Service is one of: twilio, smtp, ntfy
	Service string `point:"service"`
	// SID is the Twilio account SID
	SID string `point:"sid"`
	// AuthToken is the Twilio auth token, the SMTP password, or the ntfy
	// access token
	AuthToken string `point:"authToken"`
	// From is the Twilio from number or the SMTP from address
	From string `point:"from"`
	// Username is the SMTP auth username
	Username string `point:"username"`
	// URL is the SMTP server (host:port) or the ntfy server
	// (default https://ntfy.sh)
	URL string `point:"url"`
	// Topic is the ntfy topic
	Topic string `point:"topic"`
	Error string `point:"error"`
}

// msgServiceDedupWindow is how long a (notification ID, address) pair is
// remembered to suppress duplicate deliveries. It must be long enough to
// cover a synchronization catch-up after an outage, when the same
// notification can arrive long after it was raised.
const msgServiceDedupWindow = time.Hour

// MsgServiceClient delivers notifications through an external message
// service. Twilio and SMTP need per-user addressing, so they consume
// message points emitted by user nodes. ntfy has a global destination
// (a topic), so it consumes notification points directly and works with
// no user nodes in scope.
//
// Deliveries are deduplicated on (notification ID, destination address),
// so a user mirrored into two groups, or the same notification arriving
// by two paths, produces one delivery. The deduplication state is
// in-memory only; a restart inside the window can produce a duplicate.
type MsgServiceClient struct {
	nc               *nats.Conn
	config           MsgService
	stop             chan struct{}
	newPoints        chan NewPoints
	newEdgePoints    chan NewPoints
	newMessages      chan data.Message
	newNotifications chan data.Notification
	upSub            *nats.Subscription
	sent             map[string]time.Time
}

// NewMsgServiceClient returns a new MsgServiceClient using its
// configuration read from the Client Manager
func NewMsgServiceClient(nc *nats.Conn, config MsgService) Client {
	return &MsgServiceClient{
		nc:               nc,
		config:           config,
		stop:             make(chan struct{}),
		newPoints:        make(chan NewPoints),
		newEdgePoints:    make(chan NewPoints),
		newMessages:      make(chan data.Message),
		newNotifications: make(chan data.Notification),
		sent:             make(map[string]time.Time),
	}
}

// Run the main logic for this client and blocks until stopped
func (mc *MsgServiceClient) Run() error {
	// watch for message and notification points raised anywhere in the
	// parent's subtree
	subject := fmt.Sprintf("up.%v.>", mc.config.Parent)

	var err error
	mc.upSub, err = mc.nc.Subscribe(subject, func(natsMsg *nats.Msg) {
		points, err := data.DecodePoints(natsMsg.Data)
		if err != nil {
			log.Println("Error decoding points in msg service upSub:", err)
			return
		}

		// up.<parentId>.<nodeId>.<type>.<key> = 5 chunks for node points
		chunks := strings.Split(natsMsg.Subject, ".")
		if len(chunks) != 5 {
			return
		}

		for _, p := range points {
			if p.Tombstone%2 != 0 {
				continue
			}

			switch p.Type {
			case data.PointTypeMessage:
				m, err := data.PointToMessage(p)
				if err != nil {
					log.Println("Msg service error decoding message:", err)
					continue
				}
				mc.newMessages <- m

			case data.PointTypeNotification:
				n, err := data.PointToNotification(p)
				if err != nil {
					log.Println("Msg service error decoding notification:", err)
					continue
				}
				mc.newNotifications <- n
			}
		}
	})

	if err != nil {
		return fmt.Errorf("Msg service error subscribing to upsub: %w", err)
	}

	sweepTicker := time.NewTicker(msgServiceDedupWindow / 4)
	defer sweepTicker.Stop()

done:
	for {
		select {
		case <-mc.stop:
			break done

		case m := <-mc.newMessages:
			switch mc.config.Service {
			case data.PointValueTwilio:
				if m.Phone == "" {
					continue
				}
				mc.deliver(m.NotificationID, m.Phone, func() error {
					tw := msg.NewTwilio(mc.config.SID, mc.config.AuthToken,
						mc.config.From)
					return tw.SendSMS(m.Phone, m.Message)
				})
			case data.PointValueSMTP:
				if m.Email == "" {
					continue
				}
				mc.deliver(m.NotificationID, m.Email, func() error {
					sm := msg.NewSMTP(mc.config.URL, mc.config.Username,
						mc.config.AuthToken, mc.config.From)
					return sm.Send(m.Email, m.Subject, m.Message)
				})
			}

		case n := <-mc.newNotifications:
			switch mc.config.Service {
			case data.PointValueNtfy:
				address := mc.config.URL + "/" + mc.config.Topic
				mc.deliver(n.ID, address, func() error {
					nt := msg.NewNtfy(mc.config.URL, mc.config.Topic,
						mc.config.AuthToken)
					return nt.Send(n.Subject, n.Message)
				})
			}

		case <-sweepTicker.C:
			for k, t := range mc.sent {
				if time.Since(t) > msgServiceDedupWindow {
					delete(mc.sent, k)
				}
			}

		case pts := <-mc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &mc.config)
			if err != nil {
				log.Println("error merging msg service points:", err)
			}

		case pts := <-mc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points,
				&mc.config)
			if err != nil {
				log.Println("error merging msg service edge points:", err)
			}
		}
	}

	return mc.upSub.Unsubscribe()
}

// deliver sends one message through the service unless the same
// notification has already been delivered to the same address inside the
// deduplication window. Failed sends are not recorded, so a later copy of
// the notification arriving by another path retries the delivery.
func (mc *MsgServiceClient) deliver(notificationID, address string,
	send func() error) {
	key := ""
	if notificationID != "" {
		key = notificationID + "|" + address
		if t, ok := mc.sent[key]; ok &&
			time.Since(t) <= msgServiceDedupWindow {
			return
		}
	}

	err := send()
	if err != nil {
		mc.processError(fmt.Sprintf("Error sending to %v: %v", address, err))
		return
	}

	if key != "" {
		mc.sent[key] = time.Now()
	}

	mc.processError("")
}

// processError reports delivery failures on the error point of the
// service node, and clears it when a delivery succeeds
func (mc *MsgServiceClient) processError(errS string) {
	if errS == mc.config.Error {
		return
	}

	if errS != "" {
		log.Printf("Msg service %v: %v\n", mc.config.Description, errS)
	}

	p := data.NewPointString(data.PointTypeError, "", errS)
	p.Time = time.Now()

	err := SendNodePoint(mc.nc, mc.config.ID, p, false)
	if err != nil {
		log.Println("Msg service error sending error point:", err)
	} else {
		mc.config.Error = errS
	}
}

// Stop sends a signal to the Run function to exit
func (mc *MsgServiceClient) Stop(_ error) {
	close(mc.stop)
}

// Points is called by the Manager when new points for this node are
// received.
func (mc *MsgServiceClient) Points(nodeID string, points []data.Point) {
	mc.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this node
// are received.
func (mc *MsgServiceClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	mc.newEdgePoints <- NewPoints{nodeID, parentID, points}
}
