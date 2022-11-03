package node

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// Upstream is used to manage an upstream connection (cloud, etc)
type Upstream struct {
	nc                 *nats.Conn
	node               data.NodeEdge
	nodeUp             *UpstreamNode
	uri                string
	ncUp               *nats.Conn
	subUpNodePoints    map[string]*nats.Subscription
	subUpEdgePoints    map[string]*nats.Subscription
	subLocalNodePoints *nats.Subscription
	subLocalEdgePoints *nats.Subscription
	lock               sync.Mutex
	closeSync          chan bool
}

// NewUpstream is used to create a new upstream connection
func NewUpstream(nc *nats.Conn, node data.NodeEdge) (*Upstream, error) {
	var err error

	up := &Upstream{
		nc:              nc,
		node:            node,
		subUpNodePoints: make(map[string]*nats.Subscription),
		subUpEdgePoints: make(map[string]*nats.Subscription),
		closeSync:       make(chan bool),
	}

	up.nodeUp, err = NewUpstreamNode(node)
	if err != nil {
		return nil, err
	}

	if up.nodeUp.Disabled {
		log.Printf("Upstream %v disabled", up.nodeUp.Description)
		return up, nil
	}

	opts := client.EdgeOptions{
		URI:       up.nodeUp.URI,
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
		},
	}

	up.ncUp, err = client.EdgeConnect(opts)

	if err != nil {
		return nil, fmt.Errorf("Error connection to upstream NATS: %v", err)
	}

	up.subLocalNodePoints, err = nc.Subscribe(client.SubjectNodeAllPoints(), func(msg *nats.Msg) {
		nodeID, points, err := client.DecodeNodePointsMsg(msg)

		if err != nil {
			log.Println("Error decoding point: ", err)
			return
		}

		err = client.SendNodePoints(up.ncUp, nodeID, points, false)

		if err != nil {
			log.Println("Error sending node points to remote system: ", err)
		}
	})

	up.subLocalEdgePoints, err = nc.Subscribe(client.SubjectEdgeAllPoints(), func(msg *nats.Msg) {
		nodeID, parentID, points, err := client.DecodeEdgePointsMsg(msg)

		if err != nil {
			log.Println("Error decoding point: ", err)
			return
		}

		err = client.SendEdgePoints(up.ncUp, nodeID, parentID, points, false)

		if err != nil {
			log.Println("Error sending edge points to remote system: ", err)
		}

		// if point contains a tombstone value, something may have been
		// created, so watch the upstream node
		for _, p := range points {
			if p.Type == data.PointTypeTombstone {
				err := up.addUpstreamNodeSub(nodeID)
				if err != nil {
					log.Printf("Error adding upstream node sub: %v\n", err)
				}

				err = up.addUpstreamEdgeSub(nodeID, parentID)
				if err != nil {
					log.Printf("Error adding upstream edge sub: %v\n", err)
				}
			}
		}
	})

	rootNodes, err := client.GetNodes(nc, "root", "all", "", false)

	if err != nil {
		return nil, err
	}

	if len(rootNodes) == 0 {
		return nil, errors.New("root node not found")
	}

	var rootNode = rootNodes[0]

	var watchNode func(node data.NodeEdge) error

	watchNode = func(node data.NodeEdge) error {
		err := up.addUpstreamSub(node)
		if err != nil {
			return fmt.Errorf("Failed to add upstream sub: %v", err)
		}

		childNodes, err := client.GetNodes(nc, node.ID, "all", "", false)
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
	go func(ch chan bool) {
		timer := time.NewTimer(time.Millisecond * 10)

		for {
			select {
			case <-timer.C:
				err := up.syncNode(rootNode.ID, "none")
				if err != nil {
					fmt.Printf("Error syncing: %v\n", err)
				}
				timer.Reset(time.Second * 10)
			case <-ch:
				fmt.Println("Stopping sync for ", up.nodeUp.Description)
				return
			}
		}
	}(up.closeSync)

	return up, nil
}

func (up *Upstream) addUpstreamSub(node data.NodeEdge) error {
	err := up.addUpstreamNodeSub(node.ID)
	if err != nil {
		return fmt.Errorf("Error adding upstream node sub: %v", err)
	}

	err = up.addUpstreamEdgeSub(node.ID, node.Parent)
	if err != nil {
		return fmt.Errorf("Error adding upstream edge sub: %v", err)
	}

	return nil
}

func (up *Upstream) addUpstreamNodeSub(nodeID string) error {
	// check if subscriptional already exists
	up.lock.Lock()
	_, ok := up.subUpNodePoints[nodeID]
	up.lock.Unlock()
	if ok {
		// subscription allready exists
		return nil
	}

	// create subscription
	subject := client.SubjectNodePoints(nodeID)
	sub, err := up.ncUp.Subscribe(subject, func(msg *nats.Msg) {
		nodeID, points, err := client.DecodeNodePointsMsg(msg)

		if err != nil {
			log.Println("Error decoding point: ", err)
			return
		}

		err = client.SendNodePoints(up.nc, nodeID, points, false)

		if err != nil {
			log.Println("Error sending points to local system: ", err)
		}
	})

	if err != nil {
		return err
	}

	up.lock.Lock()
	up.subUpNodePoints[nodeID] = sub
	up.lock.Unlock()

	return nil
}

func (up *Upstream) addUpstreamEdgeSub(nodeID, parentID string) error {
	if nodeID == "" || parentID == "" {
		// the root node does not have an edge id
		return nil
	}

	key := nodeID + ":" + parentID

	// check if subscriptional already exists
	up.lock.Lock()
	_, ok := up.subUpEdgePoints[key]
	up.lock.Unlock()
	if ok {
		// subscription allready exists
		return nil
	}

	// create subscription
	subject := client.SubjectEdgePoints(nodeID, parentID)
	sub, err := up.ncUp.Subscribe(subject, func(msg *nats.Msg) {
		nodeID, parentID, points, err := client.DecodeEdgePointsMsg(msg)

		if err != nil {
			log.Println("Error decoding point: ", err)
			return
		}

		err = client.SendEdgePoints(up.nc, nodeID, parentID, points, false)

		if err != nil {
			log.Println("Error sending edge points to local system: ", err)
		}
	})

	if err != nil {
		return err
	}

	up.lock.Lock()
	up.subUpEdgePoints[key] = sub
	up.lock.Unlock()

	return nil
}

// sendNodesUp is used to send node and children over nats
// from one NATS server to another. Typically from the current instance
// to an upstream.
func (up *Upstream) sendNodesUp(node data.NodeEdge) error {
	err := client.SendNode(up.ncUp, node, up.node.ID)

	if err != nil {
		return err
	}

	// process child nodes
	childNodes, err := client.GetNodes(up.nc, node.ID, "all", "", false)
	if err != nil {
		return fmt.Errorf("Error getting node children: %v", err)
	}

	for _, childNode := range childNodes {
		err := up.sendNodesUp(childNode)

		if err != nil {
			return fmt.Errorf("Error sending child node: %v", err)
		}
	}

	return nil
}

func (up *Upstream) syncNode(id, parent string) error {
	nodeLocals, err := client.GetNodes(up.nc, parent, id, "", true)
	if err != nil {
		return fmt.Errorf("Error getting local node: %v", err)
	}

	if len(nodeLocals) == 0 {
		return errors.New("local nodes not found")
	}

	nodeLocal := nodeLocals[0]

	nodeUps, upErr := client.GetNodes(up.ncUp, parent, id, "", true)
	if upErr != nil {
		if upErr != data.ErrDocumentNotFound {
			return fmt.Errorf("Error getting upstream root node: %v", upErr)
		}
	}

	var nodeUp data.NodeEdge

	if len(nodeUps) == 0 || upErr == data.ErrDocumentNotFound {
		log.Printf("Upstream node %v does not exist, sending\n", nodeLocal.Desc())
		err := up.sendNodesUp(nodeLocal)
		if err != nil {
			return fmt.Errorf("Error sending node upstream: %w", err)
		}

		err = up.addUpstreamSub(nodeLocal)
		if err != nil {
			log.Println("Error subscribing to upstream node: ", err)
		}
	} else {
		nodeUp = nodeUps[0]
	}

	if nodeUp.Hash == nodeLocal.Hash {
		log.Printf("syncing node: %v, hash up: 0x%x, down: 0x%x ",
			nodeLocal.Desc(),
			nodeUp.Hash, nodeLocal.Hash)

		// first compare node points
		// key in below map is the index of the point in the upstream node
		upstreamProcessed := make(map[int]bool)

		for _, p := range nodeLocal.Points {
			found := false
			for i, pUp := range nodeUp.Points {
				if p.IsMatch(pUp.Type, pUp.Key) {
					found = true
					upstreamProcessed[i] = true
					if p.Time.After(pUp.Time) {
						// need to send point upstream
						err := client.SendNodePoint(up.ncUp, nodeUp.ID, p, true)
						if err != nil {
							log.Println("Error syncing point upstream: ", err)
						}
					} else if p.Time.Before(pUp.Time) {
						// need to update point locally
						err := client.SendNodePoint(up.nc, nodeLocal.ID, pUp, true)
						if err != nil {
							log.Println("Error syncing point from upstream: ", err)
						}
					}
				}
			}

			if !found {
				client.SendNodePoint(up.ncUp, nodeUp.ID, p, true)
			}
		}

		// check for any points that do not exist locally
		for i, pUp := range nodeUp.Points {
			if _, ok := upstreamProcessed[i]; !ok {
				err := client.SendNodePoint(up.nc, nodeLocal.ID, pUp, true)
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
				if p.IsMatch(pUp.Type, pUp.Key) {
					found = true
					upstreamProcessed[i] = true
					if p.Time.After(pUp.Time) {
						// need to send point upstream
						err := client.SendEdgePoint(up.ncUp, nodeUp.ID, nodeUp.Parent, p, true)
						if err != nil {
							log.Println("Error syncing point upstream: ", err)
						}
					} else if p.Time.Before(pUp.Time) {
						// need to update point locally
						err := client.SendEdgePoint(up.nc, nodeLocal.ID, nodeLocal.Parent, pUp, true)
						if err != nil {
							log.Println("Error syncing point from upstream: ", err)
						}
					}
				}
			}

			if !found {
				client.SendEdgePoint(up.ncUp, nodeUp.ID, nodeUp.Parent, p, true)
			}
		}

		// check for any points that do not exist locally
		for i, pUp := range nodeUp.EdgePoints {
			if _, ok := upstreamProcessed[i]; !ok {
				err := client.SendEdgePoint(up.nc, nodeLocal.ID, nodeLocal.Parent, pUp, true)
				if err != nil {
					log.Println("Error syncing edge point from upstream: ", err)
				}
			}
		}

		// sync child nodes
		children, err := client.GetNodes(up.nc, nodeLocal.ID, "all", "", false)
		if err != nil {
			return fmt.Errorf("Error getting local node children: %v", err)
		}

		// FIXME optimization we get the edges here and not the full child node
		upChildren, err := client.GetNodes(up.ncUp, nodeUp.ID, "all", "", false)
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
					if child.Hash != upChild.Hash {
						err := up.syncNode(child.ID, nodeLocal.ID)
						if err != nil {
							fmt.Println("Error syncing node: ", err)
						}
					}
				}
			}

			if !found {
				// need to send node upstream
				err := up.sendNodesUp(child)

				if err != nil {
					log.Println("Error sending node upstream: ", err)
				}

				err = up.addUpstreamSub(child)
				if err != nil {
					log.Println("Error subscribing to upstream node: ", err)
				}
			}
		}

		for i, upChild := range upChildren {
			if _, ok := upChildProcessed[i]; !ok {
				err := up.sendNodesUp(upChild)
				if err != nil {
					log.Println("Error getting node from upstream: ", err)
				}

				err = up.addUpstreamSub(upChild)
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
	if up.nodeUp.Disabled {
		return
	}

	if up.subLocalNodePoints != nil {
		err := up.subLocalNodePoints.Unsubscribe()
		if err != nil {
			log.Println("Error unsubscribing node points from local bus: ", err)
		}
	}

	if up.subLocalEdgePoints != nil {
		err := up.subLocalEdgePoints.Unsubscribe()
		if err != nil {
			log.Println("Error unsubscribing edge points from local bus: ", err)
		}
	}

	up.lock.Lock()
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
	up.lock.Unlock()

	up.closeSync <- true

	if up.ncUp != nil {
		up.ncUp.Close()
	}
}
