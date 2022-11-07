package client

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Upstream represents an upstream node config
type Upstream struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	URI         string `point:"uri"`
	AuthToken   string `point:"authToken"`
	Disabled    bool   `point:"disabled"`
}

// UpstreamClient is a SIOT client used to handle upstream connections
type UpstreamClient struct {
	nc                  *nats.Conn
	config              Upstream
	stop                chan struct{}
	newPoints           chan NewPoints
	newEdgePoints       chan NewPoints
	ncRemote            *nats.Conn
	subRemoteNodePoints *nats.Subscription
	subRemoteEdgePoints *nats.Subscription
	subLocalNodePoints  *nats.Subscription
	subLocalEdgePoints  *nats.Subscription
	chConnected         chan bool
	// FIXME: can we get rid of lock?
	lock sync.Mutex
}

// NewUpstreamClient constructor
func NewUpstreamClient(nc *nats.Conn, config Upstream) Client {
	return &UpstreamClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		chConnected:   make(chan bool),
	}
}

// GetNodes has a 20s timeout, so lets use that here
var syncTimeout = 20 * time.Second

// Start runs the main logic for this client and blocks until stopped
func (up *UpstreamClient) Start() error {
	// FIXME: determine what sync interval we want
	syncTicker := time.NewTicker(time.Second * 10)
	syncTicker.Stop()

	connectTimer := time.NewTimer(time.Millisecond * 10)

	rootNode, err := GetRootNode(up.nc)
	if err != nil {
		return fmt.Errorf("Error getting root node: %v", err)
	}

done:
	for {
		select {
		case <-up.stop:
			log.Println("Stopping upstream client: ", up.config.Description)
			break done
		case <-connectTimer.C:
			err := up.connect()
			if err != nil {
				log.Printf("BUG, this should never happy: Error connecting upstream %v: %v\n",
					up.config.Description, err)
				connectTimer.Reset(30 * time.Second)
			}
		case <-syncTicker.C:
			err := up.syncNode(rootNode.ID, "none")
			if err != nil {
				log.Println("Error syncing: ", err)
			}
		case conn := <-up.chConnected:
			if conn {
				syncTicker.Reset(syncTimeout)
				err := up.syncNode(rootNode.ID, "none")
				if err != nil {
					log.Println("Error syncing: ", err)
				}
			} else {
				syncTicker.Stop()
			}
		case pts := <-up.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &up.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeURI,
					data.PointTypeAuthToken,
					data.PointTypeDisable:
					// we need to restart the influx write API
					up.disconnect()
					connectTimer.Reset(10 * time.Millisecond)
				}
			}

		case pts := <-up.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &up.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
		}
	}

	// clean up
	up.disconnect()

	return nil
}

// Stop sends a signal to the Start function to exit
func (up *UpstreamClient) Stop(err error) {
	close(up.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (up *UpstreamClient) Points(nodeID string, points []data.Point) {
	up.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (up *UpstreamClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	up.newEdgePoints <- NewPoints{nodeID, parentID, points}
}

func (up *UpstreamClient) connect() error {
	if up.config.Disabled {
		log.Printf("Upstream %v disabled", up.config.Description)
		return nil
	}

	opts := EdgeOptions{
		URI:       up.config.URI,
		AuthToken: up.config.AuthToken,
		NoEcho:    true,
		Connected: func() {
			up.chConnected <- true
			log.Println("NATS Upstream Connected")
		},
		Disconnected: func() {
			up.chConnected <- false
			log.Println("NATS Upstream Disconnected")
		},
		Reconnected: func() {
			up.chConnected <- true
			log.Println("NATS Upstream Reconnected")
		},
		Closed: func() {
			log.Println("NATS Upstream Closed")
		},
	}

	var err error
	up.ncRemote, err = EdgeConnect(opts)

	if err != nil {
		return fmt.Errorf("Error connection to upstream NATS: %v", err)
	}

	up.subLocalNodePoints, err = up.nc.Subscribe("up.*.*", func(msg *nats.Msg) {
		_, nodeID, points, err := DecodeUpNodePointsMsg(msg)

		if err != nil {
			log.Println("Error decoding point: ", err)
			return
		}

		err = SendNodePoints(up.ncRemote, nodeID, points, false)

		if err != nil {
			log.Println("Error sending node points to remote system: ", err)
		}
	})

	up.subLocalEdgePoints, err = up.nc.Subscribe("up.*.*.*", func(msg *nats.Msg) {
		_, nodeID, parentID, points, err := DecodeUpEdgePointsMsg(msg)

		if err != nil {
			log.Println("Error decoding point: ", err)
			return
		}

		err = SendEdgePoints(up.ncRemote, nodeID, parentID, points, false)

		if err != nil {
			log.Println("Error sending edge points to remote system: ", err)
		}
	})

	up.subRemoteNodePoints, err = up.ncRemote.Subscribe("up.*.*", func(msg *nats.Msg) {
		_, nodeID, points, err := DecodeUpNodePointsMsg(msg)

		if err != nil {
			log.Println("Error decoding point: ", err)
			return
		}

		err = SendNodePoints(up.ncRemote, nodeID, points, false)

		if err != nil {
			log.Println("Error sending node points to remote system: ", err)
		}
	})

	up.subRemoteEdgePoints, err = up.ncRemote.Subscribe("up.*.*.*", func(msg *nats.Msg) {
		_, nodeID, parentID, points, err := DecodeUpEdgePointsMsg(msg)

		if err != nil {
			log.Println("Error decoding point: ", err)
			return
		}

		err = SendEdgePoints(up.ncRemote, nodeID, parentID, points, false)

		if err != nil {
			log.Println("Error sending edge points to remote system: ", err)
		}
	})

	return nil
}

func (up *UpstreamClient) disconnect() {
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

	if up.ncRemote != nil {
		up.ncRemote.Close()
	}
}

// sendNodesUp is used to send node and children over nats
// from one NATS server to another. Typically from the current instance
// to an upstream.
func (up *UpstreamClient) sendNodesUp(node data.NodeEdge) error {
	err := SendNode(up.ncRemote, node, up.config.ID)

	if err != nil {
		return err
	}

	// process child nodes
	childNodes, err := GetNodes(up.nc, node.ID, "all", "", false)
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

func (up *UpstreamClient) syncNode(id, parent string) error {
	nodeLocals, err := GetNodes(up.nc, parent, id, "", true)
	if err != nil {
		return fmt.Errorf("Error getting local node: %v", err)
	}

	if len(nodeLocals) == 0 {
		return errors.New("local nodes not found")
	}

	nodeLocal := nodeLocals[0]

	nodeUps, upErr := GetNodes(up.ncRemote, parent, id, "", true)
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
						err := SendNodePoint(up.ncRemote, nodeUp.ID, p, true)
						if err != nil {
							log.Println("Error syncing point upstream: ", err)
						}
					} else if p.Time.Before(pUp.Time) {
						// need to update point locally
						err := SendNodePoint(up.nc, nodeLocal.ID, pUp, true)
						if err != nil {
							log.Println("Error syncing point from upstream: ", err)
						}
					}
				}
			}

			if !found {
				SendNodePoint(up.ncRemote, nodeUp.ID, p, true)
			}
		}

		// check for any points that do not exist locally
		for i, pUp := range nodeUp.Points {
			if _, ok := upstreamProcessed[i]; !ok {
				err := SendNodePoint(up.nc, nodeLocal.ID, pUp, true)
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
						err := SendEdgePoint(up.ncRemote, nodeUp.ID, nodeUp.Parent, p, true)
						if err != nil {
							log.Println("Error syncing point upstream: ", err)
						}
					} else if p.Time.Before(pUp.Time) {
						// need to update point locally
						err := SendEdgePoint(up.nc, nodeLocal.ID, nodeLocal.Parent, pUp, true)
						if err != nil {
							log.Println("Error syncing point from upstream: ", err)
						}
					}
				}
			}

			if !found {
				SendEdgePoint(up.ncRemote, nodeUp.ID, nodeUp.Parent, p, true)
			}
		}

		// check for any points that do not exist locally
		for i, pUp := range nodeUp.EdgePoints {
			if _, ok := upstreamProcessed[i]; !ok {
				err := SendEdgePoint(up.nc, nodeLocal.ID, nodeLocal.Parent, pUp, true)
				if err != nil {
					log.Println("Error syncing edge point from upstream: ", err)
				}
			}
		}

		// sync child nodes
		children, err := GetNodes(up.nc, nodeLocal.ID, "all", "", false)
		if err != nil {
			return fmt.Errorf("Error getting local node children: %v", err)
		}

		// FIXME optimization we get the edges here and not the full child node
		upChildren, err := GetNodes(up.ncRemote, nodeUp.ID, "all", "", false)
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
			}
		}
		for i, upChild := range upChildren {
			if _, ok := upChildProcessed[i]; !ok {
				err := up.sendNodesUp(upChild)
				if err != nil {
					log.Println("Error getting node from upstream: ", err)
				}
			}
		}
	}

	return nil
}
