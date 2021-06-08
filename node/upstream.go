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
		err := up.addUpstreamSub(node.ID)
		if err != nil {
			up.Stop()
			return nil, fmt.Errorf("Failed to add upstream sub: %v", err)
		}
	}

	// occasionally sync nodes
	go func() {
		fetchedOnce := false

		for {
			if fetchedOnce {
				time.Sleep(time.Second * 10)
			}

			fetchedOnce = true

			up.syncNode(rootID, "")
		}
	}()

	return up, nil
}

func (up *Upstream) addUpstreamSub(nodeID string) error {
	subject := nats.SubjectNodePoints(nodeID)
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
		return err
	}

	up.subUps = append(up.subUps, sub)

	return nil
}

func (up *Upstream) syncNode(id, parent string) error {
	nodeLocal, err := nats.GetNode(up.nc, id, parent)
	if err != nil {
		return fmt.Errorf("Error getting local node: %v", err)
	}

	nodeUp, err := nats.GetNode(up.ncUp, id, parent)
	if err != nil {
		return fmt.Errorf("Error getting upstream root node: %v", err)
	}

	if nodeUp.ID == "" {
		log.Printf("Upstream node %v does not exist, sending\n", nodeLocal.Desc())
		return nats.SendNode(up.nc, up.ncUp, id, "")
	}

	if nodeUp.Tombstone != nodeLocal.Tombstone {
		err := nats.SendPoint(up.ncUp, nodeUp.ID, data.Point{
			Type: data.PointTypeRemoveParent,
			Text: parent,
		}, true)

		if err != nil {
			log.Println("Error setting tombstone setting upstream: ", err)
		}
	}

	if bytes.Compare(nodeUp.Hash, nodeLocal.Hash) != 0 {
		log.Println("syncing node: ", nodeLocal.Desc())

		// first compare points
		// key in below map is the index of the point in the upstream node
		upstreamProcessed := make(map[int]bool)

		for _, p := range nodeLocal.Points {
			found := false
			for i, pUp := range nodeUp.Points {
				if p.IsMatch(pUp.ID, pUp.Type, pUp.Index) {
					found = true
					upstreamProcessed[i] = true
					if p.Time.After(pUp.Time) {
						// need to send point upstream
						err := nats.SendPoint(up.ncUp, nodeUp.ID, p, true)
						if err != nil {
							log.Println("Error syncing point upstream: ", err)
						}
					} else if p.Time.Before(pUp.Time) {
						// need to update point locally
						err := nats.SendPoint(up.nc, nodeLocal.ID, pUp, true)
						if err != nil {
							log.Println("Error syncing point from upstream: ", err)
						}
					}
				}
			}

			if !found {
				nats.SendPoint(up.ncUp, nodeUp.ID, p, true)
			}
		}

		// check for any points that do not exist locally
		for i, pUp := range nodeUp.Points {
			if _, ok := upstreamProcessed[i]; !ok {
				err := nats.SendPoint(up.nc, nodeLocal.ID, pUp, true)
				if err != nil {
					log.Println("Error syncing point from upstream: ", err)
				}

			}
		}

		// sync child nodes
		children, err := nats.GetNodeChildren(up.nc, nodeLocal.ID)
		if err != nil {
			return fmt.Errorf("Error getting local node children: %v", err)
		}

		upChildren, err := nats.GetNodeChildren(up.ncUp, nodeUp.ID)
		if err != nil {
			return fmt.Errorf("Error getting upstream node children: %v", err)
		}

		// map index is index of upChildren
		upChildProcessed := make(map[int]bool)

		for _, child := range children {
			found := false
			for i, upChild := range upChildren {
				if child.ID == upChild.ID {
					found = true
					upChildProcessed[i] = true
					if bytes.Compare(child.Hash, upChild.Hash) != 0 ||
						child.Tombstone != upChild.Tombstone {
						err := up.syncNode(child.ID, nodeLocal.ID)
						if err != nil {
							fmt.Println("Error syncing node: ", err)
						}
					}
				}
			}

			if !found {
				// need to send node upstream
				err := nats.SendNode(up.nc, up.ncUp, child.ID, nodeLocal.ID)

				if err != nil {
					log.Println("Error sending node upstream: ", err)
				}

				err = up.addUpstreamSub(nodeLocal.ID)
				if err != nil {
					log.Println("Error subscribing to upstream node: ", err)
				}
			}
		}

		for i, upChild := range upChildren {
			if _, ok := upChildProcessed[i]; !ok {
				err := nats.SendNode(up.ncUp, up.nc, upChild.ID, nodeUp.ID)
				if err != nil {
					log.Println("Error getting node from upstream: ", err)
				}

				err = up.addUpstreamSub(nodeLocal.ID)
				if err != nil {
					log.Println("Error subscribing to upstream node: ", err)
				}
			}
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
