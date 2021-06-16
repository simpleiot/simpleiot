package node

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/nats"
)

// Upstream is used to manage an upstream connection (cloud, etc)
type Upstream struct {
	nc              *natsgo.Conn
	node            data.NodeEdge
	nodeUp          *UpstreamNode
	uri             string
	ncUp            *natsgo.Conn
	subUpNodePoints map[string]*natsgo.Subscription
	subUpEdgePoints map[string]*natsgo.Subscription
	subLocal        *natsgo.Subscription
}

// NewUpstream is used to create a new upstream connection
func NewUpstream(nc *natsgo.Conn, node data.NodeEdge) (*Upstream, error) {
	var err error

	up := &Upstream{
		nc:              nc,
		node:            node,
		subUpNodePoints: make(map[string]*natsgo.Subscription),
		subUpEdgePoints: make(map[string]*natsgo.Subscription),
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

		err = nats.SendNodePoints(up.ncUp, nodeID, points, false)

		if err != nil {
			log.Println("Error sending points to remote system: ", err)
		}
	})

	rootNode, err := nats.GetNode(nc, "root", "")

	if err != nil {
		return nil, err
	}

	var watchNode func(node data.NodeEdge) error

	watchNode = func(node data.NodeEdge) error {
		err := up.addUpstreamSub(node)
		if err != nil {
			return fmt.Errorf("Failed to add upstream sub: %v", err)
		}

		childNodes, err := nats.GetNodeChildren(nc, node.ID)
		if err != nil {
			return err
		}

		for _, child := range childNodes {
			err := watchNode(child)
			if err != nil {
				return err
			}
		}

		return nil
	}

	err = watchNode(rootNode)

	if err != nil {
		up.Stop()
		return nil, fmt.Errorf("failed to watch nodes: %v", err)
	}

	// occasionally sync nodes
	go func() {
		fetchedOnce := false

		for {
			if fetchedOnce {
				time.Sleep(time.Second * 10)
			}

			fetchedOnce = true

			up.syncNode(rootNode.ID, "")
		}
	}()

	return up, nil
}

func (up *Upstream) addUpstreamSub(node data.NodeEdge) error {
	err := up.addUpstreamNodeSub(node.ID)
	if err != nil {
		return fmt.Errorf("Error adding upstream node sub: %v", err)
	}

	err = up.addUpstreamEdgeSub(node.EdgeID)
	if err != nil {
		return fmt.Errorf("Error adding upstream edge sub: %v", err)
	}

	return nil
}

func (up *Upstream) addUpstreamNodeSub(nodeID string) error {
	// check if subscriptional already exists
	_, ok := up.subUpNodePoints[nodeID]
	if ok {
		// subscription allready exists
		return nil
	}

	// create subscription
	subject := nats.SubjectNodePoints(nodeID)
	sub, err := up.ncUp.Subscribe(subject, func(msg *natsgo.Msg) {
		nodeID, points, err := nats.DecodeNodePointsMsg(msg)

		if err != nil {
			log.Println("Error decoding point: ", err)
			return
		}

		err = nats.SendNodePoints(up.nc, nodeID, points, false)

		if err != nil {
			log.Println("Error sending points to local system: ", err)
		}
	})

	if err != nil {
		return err
	}

	up.subUpNodePoints[nodeID] = sub

	return nil
}

func (up *Upstream) addUpstreamEdgeSub(edgeID string) error {
	if edgeID == "" {
		// the root node does not have an edge id
		return nil
	}
	// check if subscriptional already exists
	_, ok := up.subUpEdgePoints[edgeID]
	if ok {
		// subscription allready exists
		return nil
	}

	// create subscription
	subject := nats.SubjectEdgePoints(edgeID)
	sub, err := up.ncUp.Subscribe(subject, func(msg *natsgo.Msg) {
		edgeID, points, err := nats.DecodeNodePointsMsg(msg)

		if err != nil {
			log.Println("Error decoding point: ", err)
			return
		}

		err = nats.SendEdgePoints(up.nc, edgeID, points, false)

		if err != nil {
			log.Println("Error sending edge points to local system: ", err)
		}
	})

	if err != nil {
		return err
	}

	up.subUpEdgePoints[edgeID] = sub

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

	if bytes.Compare(nodeUp.Hash, nodeLocal.Hash) != 0 {
		log.Println("syncing node: ", nodeLocal.Desc())

		// first compare node points
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
						err := nats.SendNodePoint(up.ncUp, nodeUp.ID, p, true)
						if err != nil {
							log.Println("Error syncing point upstream: ", err)
						}
					} else if p.Time.Before(pUp.Time) {
						// need to update point locally
						err := nats.SendNodePoint(up.nc, nodeLocal.ID, pUp, true)
						if err != nil {
							log.Println("Error syncing point from upstream: ", err)
						}
					}
				}
			}

			if !found {
				nats.SendNodePoint(up.ncUp, nodeUp.ID, p, true)
			}
		}

		// check for any points that do not exist locally
		for i, pUp := range nodeUp.Points {
			if _, ok := upstreamProcessed[i]; !ok {
				err := nats.SendNodePoint(up.nc, nodeLocal.ID, pUp, true)
				if err != nil {
					log.Println("Error syncing point from upstream: ", err)
				}

			}
		}

		// now compare edge points
		// key in below map is the index of the point in the upstream node
		upstreamProcessed = make(map[int]bool)

		for _, p := range nodeLocal.EdgePoints {
			found := false
			for i, pUp := range nodeUp.EdgePoints {
				if p.IsMatch(pUp.ID, pUp.Type, pUp.Index) {
					found = true
					upstreamProcessed[i] = true
					if p.Time.After(pUp.Time) {
						// need to send point upstream
						err := nats.SendEdgePoint(up.ncUp, nodeUp.EdgeID, p, true)
						if err != nil {
							log.Println("Error syncing point upstream: ", err)
						}
					} else if p.Time.Before(pUp.Time) {
						// need to update point locally
						err := nats.SendEdgePoint(up.nc, nodeLocal.EdgeID, pUp, true)
						if err != nil {
							log.Println("Error syncing point from upstream: ", err)
						}
					}
				}
			}

			if !found {
				nats.SendEdgePoint(up.ncUp, nodeUp.EdgeID, p, true)
			}
		}

		// check for any points that do not exist locally
		for i, pUp := range nodeUp.EdgePoints {
			if _, ok := upstreamProcessed[i]; !ok {
				err := nats.SendEdgePoint(up.nc, nodeLocal.EdgeID, pUp, true)
				if err != nil {
					log.Println("Error syncing edge point from upstream: ", err)
				}
			}
		}

		// sync child nodes
		children, err := nats.GetNodeChildren(up.nc, nodeLocal.ID)
		if err != nil {
			return fmt.Errorf("Error getting local node children: %v", err)
		}

		// FIXME optimization we get the edges here and not the full child node
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
					if bytes.Compare(child.Hash, upChild.Hash) != 0 {
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

				err = up.addUpstreamSub(nodeLocal)
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

				err = up.addUpstreamSub(nodeLocal)
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

	for _, sub := range up.subUpNodePoints {
		err := sub.Unsubscribe()
		if err != nil {
			log.Println("Error unsubscribing from upstream bus: ", err)
		}
	}

	for _, sub := range up.subUpEdgePoints {
		err := sub.Unsubscribe()
		if err != nil {
			log.Println("Error unsubscribing from upstream bus: ", err)
		}
	}

	if up.ncUp != nil {
		up.ncUp.Close()
	}
}
