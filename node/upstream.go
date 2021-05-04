package node

import (
	"fmt"
	"log"
	"os"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
	"github.com/simpleiot/simpleiot/nats"
)

// Upstream is used to manage an upstream connection (cloud, etc)
type Upstream struct {
	nc       *natsgo.Conn
	db       *db.Db
	node     data.NodeEdge
	nodeUp   *UpstreamNode
	uri      string
	ncUp     *natsgo.Conn
	subUps   []*natsgo.Subscription
	subLocal *natsgo.Subscription
}

// NewUpstream is used to create a new upstream connection
func NewUpstream(db *db.Db, nc *natsgo.Conn, node data.NodeEdge) (*Upstream, error) {
	var err error

	up := &Upstream{
		nc:   nc,
		db:   db,
		node: node,
	}

	up.nodeUp, err = NewUpstreamNode(node)
	if err != nil {
		return nil, err
	}

	up.subLocal, err = nc.Subscribe(nats.SubjectNodeAllPoints(), func(msg *natsgo.Msg) {
		nodeID, points, err := nats.DecodeNodePointsMsg(msg)

		if err != nil {
			log.Println("Error decoding point: ", err)
			return
		}

		err = nats.SendPoints(up.ncUp, nodeID, points, false)

		if err != nil {
			log.Println("Error sending points to remote system: ", err)
		}
	})

	opts := nats.EdgeOptions{
		Server:    up.nodeUp.URI,
		AuthToken: up.nodeUp.AuthToken,
		NoEcho:    true,
		Disconnected: func() {
			log.Println("NATS Upstream Disconnected")
		},
		Reconnected: func() {
			log.Println("NATS Upstream Reconnected")
		},
		Closed: func() {
			log.Println("NATS Upstream Closed")
			os.Exit(0)
		},
	}

	up.ncUp, err = nats.EdgeConnect(opts)

	if err != nil {
		return nil, fmt.Errorf("Error connection to upstream NATS: %v", err)
	}

	rootID := up.db.RootNodeID()
	nodes, err := up.db.NodeDescendents(rootID, "", true)

	for _, node := range nodes {
		subject := nats.SubjectNodePoints(node.ID)
		sub, err := up.ncUp.Subscribe(subject, func(msg *natsgo.Msg) {
			nodeID, points, err := nats.DecodeNodePointsMsg(msg)

			if err != nil {
				log.Println("Error decoding point: ", err)
				return
			}

			err = nats.SendPoints(up.nc, nodeID, points, false)

			if err != nil {
				log.Println("Error sending points to local system: ", err)
			}
		})

		if err != nil {
			up.Stop()
			return nil, err
		}

		up.subUps = append(up.subUps, sub)

	}

	return up, nil
}

// Stop upstream instance
func (up *Upstream) Stop() {
	if up.subLocal != nil {
		err := up.subLocal.Unsubscribe()
		if err != nil {
			log.Println("Error unsubscribing from local bus: ", err)
		}
	}

	for _, sub := range up.subUps {
		err := sub.Unsubscribe()
		if err != nil {
			log.Println("Error unsubscribing from upstream bus: ", err)
		}
	}

	if up.ncUp != nil {
		up.ncUp.Close()
	}
}
