package api

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	natsgo "github.com/nats-io/nats.go"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db/genji"
)

// NatsHandler implements the SIOT NATS api
type NatsHandler struct {
	server    string
	Nc        *nats.Conn
	db        *genji.Db
	authToken string
	lock      sync.Mutex
	updates   map[string]time.Time
}

// NewNatsHandler creates a new NATS client for handling SIOT requests
func NewNatsHandler(db *genji.Db, authToken, server string) *NatsHandler {
	log.Println("NATS handler connecting to: ", server)
	return &NatsHandler{
		db:        db,
		authToken: authToken,
		updates:   make(map[string]time.Time),
		server:    server,
	}
}

// Connect to NATS server and set up handlers for things we are interested in
func (nh *NatsHandler) Connect() (*natsgo.Conn, error) {
	nc, err := nats.Connect(nh.server,
		nats.Timeout(10*time.Second),
		nats.PingInterval(60*5*time.Second),
		nats.MaxPingsOutstanding(5),
		nats.ReconnectBufSize(5*1024*1024),
		nats.SetCustomDialer(&net.Dialer{
			KeepAlive: -1,
		}),
		nats.Token(nh.authToken),
	)

	if err != nil {
		return nil, err
	}

	nh.Nc = nc

	if _, err := nc.Subscribe("node.*.samples", nh.handlePoints); err != nil {
		return nil, fmt.Errorf("Subscribe node samples error: %w", err)
	}

	if _, err := nc.Subscribe("node.*.points", nh.handlePoints); err != nil {
		return nil, fmt.Errorf("Subscribe node points error: %w", err)
	}

	return nc, nil
}

// StartUpdate starts an update
func (nh *NatsHandler) StartUpdate(id, url string) error {
	nh.lock.Lock()
	defer nh.lock.Unlock()

	if _, ok := nh.updates[id]; ok {
		return fmt.Errorf("Update already in process for dev: %v", id)
	}

	nh.updates[id] = time.Now()

	err := nh.db.NodeSetSwUpdateState(id, data.SwUpdateState{
		Running: true,
	})

	if err != nil {
		delete(nh.updates, id)
		return err
	}

	go func() {
		err := NatsSendFileFromHTTP(nh.Nc, id, url, func(bytesTx int) {
			err := nh.db.NodeSetSwUpdateState(id, data.SwUpdateState{
				Running:     true,
				PercentDone: bytesTx,
			})

			if err != nil {
				log.Println("Error setting update status in DB: ", err)
			}
		})

		state := data.SwUpdateState{
			Running: false,
		}

		if err != nil {
			state.Error = "Error updating software"
			state.PercentDone = 0
		} else {
			state.PercentDone = 100
		}

		nh.lock.Lock()
		delete(nh.updates, id)
		nh.lock.Unlock()

		err = nh.db.NodeSetSwUpdateState(id, state)
		if err != nil {
			log.Println("Error setting sw update state: ", err)
		}
	}()

	return nil
}

// FIXME consider moving this to db package and then unexporting the NodePoint method
func (nh *NatsHandler) handlePoints(msg *nats.Msg) {
	chunks := strings.Split(msg.Subject, ".")
	if len(chunks) < 3 {
		log.Println("Error decoding node samples subject: ", msg.Subject)
		nh.reply(msg.Reply, errors.New("error decoding node samples subject"))
		return
	}
	nodeID := chunks[1]
	points, err := data.PbDecodePoints(msg.Data)
	if err != nil {
		log.Println("Error decoding Pb Samples: ", err)
		nh.reply(msg.Reply, err)
		return
	}

	for _, p := range points {
		err = nh.db.NodePoint(nodeID, p)
		if err != nil {
			log.Println("Error writing point to Db: ", err)
			nh.reply(msg.Reply, err)
			return
		}
	}

	nh.reply(msg.Reply, nil)
}

// used for messages that want an ACK
func (nh *NatsHandler) reply(subject string, err error) {
	if subject == "" {
		// node is not expecting a reply
		return
	}

	reply := ""

	if err != nil {
		reply = err.Error()
	}

	nh.Nc.Publish(subject, []byte(reply))
}
