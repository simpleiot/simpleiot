package node

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"time"

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

	rootID := up.db.RootNodeID()
	nodes, err := up.db.NodeDescendents(rootID, "", true)

	// FIXME -- handle when node structure changes (add/remove nodes)
	// subscribe to remote changes for all nodes on this device
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

	// occasionally sync nodes
	go func() {
		fetchedOnce := false

		for {
			if fetchedOnce {
				time.Sleep(time.Second * 10)
			}

			fetchedOnce = true

			fmt.Println("CLIFF: syncing upstream")
			up.syncNode(rootID)
		}
	}()

	return up, nil
}

func (up *Upstream) syncNode(id string) error {
	nodeLocal, err := nats.GetNode(up.nc, id)
	if err != nil {
		return fmt.Errorf("Error getting local node: %v", err)
	}

	fmt.Printf("CLIFF: nodeLocal: %+v\n", nodeLocal)

	nodeUp, err := nats.GetNode(up.ncUp, id)
	if err != nil {
		return fmt.Errorf("Error getting upstream root node: %v", err)
	}

	if nodeUp.ID == "" {
		log.Println("Upstream node does not exist, sending: ", id)
		return up.sendNodeUpstream(id, "")
		return nil
	}

	fmt.Printf("CLIFF: nodeUp: %+v\n", nodeUp)

	if bytes.Compare(nodeUp.Hash, nodeLocal.Hash) != 0 {
		log.Println("root node hash differs")
	}

	return nil
}

// sendNodeUpstream sends complete node upstream and any child nodes
func (up *Upstream) sendNodeUpstream(id, parent string) error {
	node, err := nats.GetNode(up.nc, id)
	if err != nil {
		return fmt.Errorf("Error getting local node: %v", err)
	}

	points := node.Points

	points = append(points, data.Point{
		Type: data.PointTypeNodeType,
		Text: node.Type,
	})

	if parent != "" {
		points = append(points, data.Point{
			Type: data.PointTypeParent,
			Text: parent,
		})
	}

	err = nats.SendPoints(up.ncUp, id, points, true)

	if err != nil {
		return fmt.Errorf("Error sending node upstream: %v", err)
	}

	// process child nodes
	childNodes, err := nats.GetNodeChildren(up.nc, id)
	if err != nil {
		return fmt.Errorf("Error getting node children: %v", err)
	}

	for _, childNode := range childNodes {
		err := up.sendNodeUpstream(childNode.ID, id)

		if err != nil {
			return fmt.Errorf("Error sending child node: %v", err)
		}
	}

	return nil
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
