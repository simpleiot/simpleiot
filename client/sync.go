package client

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Sync represents an sync node config
type Sync struct {
	ID             string `node:"id"`
	Parent         string `node:"parent"`
	Description    string `point:"description"`
	URI            string `point:"uri"`
	AuthToken      string `point:"authToken"`
	Period         int    `point:"period"`
	Disable        bool   `point:"disable"`
	SyncCount      int    `point:"syncCount"`
	SyncCountReset bool   `point:"syncCountReset"`
}

type newEdge struct {
	parent string
	id     string
	local  bool
}

// SyncClient is a SIOT client used to handle upstream connections
type SyncClient struct {
	nc                  *nats.Conn
	ncLocal             *nats.Conn
	ncRemote            *nats.Conn
	rootLocal           data.NodeEdge
	rootRemote          data.NodeEdge
	config              Sync
	stop                chan struct{}
	newPoints           chan NewPoints
	newEdgePoints       chan NewPoints
	subRemoteNodePoints map[string]*nats.Subscription
	subRemoteEdgePoints map[string]*nats.Subscription
	subRemoteUp         *nats.Subscription
	chConnected         chan bool
	initialSub          bool
	chNewEdge           chan newEdge
}

// NewSyncClient constructor
func NewSyncClient(nc *nats.Conn, config Sync) Client {
	return &SyncClient{
		nc:                  nc,
		config:              config,
		stop:                make(chan struct{}),
		newPoints:           make(chan NewPoints),
		newEdgePoints:       make(chan NewPoints),
		chConnected:         make(chan bool),
		subRemoteNodePoints: make(map[string]*nats.Subscription),
		subRemoteEdgePoints: make(map[string]*nats.Subscription),
		chNewEdge:           make(chan newEdge),
	}
}

// Run the main logic for this client and blocks until stopped
func (up *SyncClient) Run() error {
	// create a new NATs connection to the local server as we need to
	// turn echo off
	uri, token, err := GetNatsURI(up.nc)
	if err != nil {
		return fmt.Errorf("Error getting NATS URI: %v", err)
	}

	opts := EdgeOptions{
		URI:       uri,
		AuthToken: token,
		NoEcho:    true,
		Connected: func() {
			log.Printf("Sync: %v: Local Connected: %v\n", up.config.Description, uri)
		},
		Disconnected: func() {
			log.Printf("Sync: %v: Local Disconnected\n", up.config.Description)
		},
		Reconnected: func() {
			log.Printf("Sync: %v: Local Reconnected\n", up.config.Description)
		},
		Closed: func() {
			log.Printf("Sync: %v: Local Closed\n", up.config.Description)
		},
	}

	up.ncLocal, err = EdgeConnect(opts)
	if err != nil {
		return fmt.Errorf("Error connection to local NATS: %v", err)
	}

	chLocalNodePoints := make(chan NewPoints)
	chLocalEdgePoints := make(chan NewPoints)

	subLocalNodePoints, err := up.ncLocal.Subscribe(SubjectNodeAllPoints(), func(msg *nats.Msg) {
		nodeID, points, err := DecodeNodePointsMsg(msg)

		if err != nil {
			log.Println("Error decoding point: ", err)
			return
		}

		chLocalNodePoints <- NewPoints{ID: nodeID, Points: points}

	})

	subLocalEdgePoints, err := up.ncLocal.Subscribe(SubjectEdgeAllPoints(), func(msg *nats.Msg) {
		nodeID, parentID, points, err := DecodeEdgePointsMsg(msg)

		if err != nil {
			log.Println("Error decoding point: ", err)
			return
		}

		chLocalEdgePoints <- NewPoints{ID: nodeID, Parent: parentID, Points: points}

		for _, p := range points {
			if p.Type == data.PointTypeTombstone && p.Value == 0 {
				// a new node was likely created, make sure we watch it
				up.chNewEdge <- newEdge{parent: parentID, id: nodeID, local: true}
			}
		}
	})

	checkPeriod := func() {
		if up.config.Period < 1 {
			up.config.Period = 20
			points := data.Points{
				{Type: data.PointTypePeriod, Value: float64(up.config.Period)},
			}

			err = SendPoints(up.nc, SubjectNodePoints(up.config.ID), points, false)
			if err != nil {
				log.Println("Error resetting sync sync count: ", err)
			}
		}
	}

	checkPeriod()

	syncTicker := time.NewTicker(time.Second * 10)
	syncTicker.Stop()

	connectTimer := time.NewTimer(time.Millisecond * 10)

	up.rootLocal, err = GetRootNode(up.nc)
	if err != nil {
		return fmt.Errorf("Error getting root node: %v", err)
	}

	connected := false
	up.initialSub = false

done:
	for {
		select {
		case <-up.stop:
			log.Println("Stopping upstream client: ", up.config.Description)
			break done
		case <-connectTimer.C:
			err := up.connect()
			if err != nil {
				log.Printf("Sync connect failure: %v: %v\n",
					up.config.Description, err)
				connectTimer.Reset(30 * time.Second)
			}
		case <-syncTicker.C:
			err := up.syncNode("root", up.rootLocal.ID)
			if err != nil {
				log.Println("Error syncing: ", err)
			}

		case conn := <-up.chConnected:
			connected = conn
			if conn {
				syncTicker.Reset(time.Duration(up.config.Period) * time.Second)
				err := up.syncNode("root", up.rootLocal.ID)
				if err != nil {
					log.Println("Error syncing: ", err)
				}

				if !up.initialSub {
					// set up initial subscriptions to remote nodes
					err = up.subscribeRemoteNode(up.rootLocal.Parent, up.rootLocal.ID)
					if err != nil {
						log.Println("Sync: initial sub failed: ", err)
					} else {
						up.initialSub = true
					}
				}
			} else {
				syncTicker.Stop()
				// the following is required in case a new server
				// is set up which may have a new root
				up.rootRemote = data.NodeEdge{}
			}
		case pts := <-chLocalNodePoints:
			if connected {
				err = SendNodePoints(up.ncRemote, pts.ID, pts.Points, false)
				if err != nil {
					log.Println("Error sending node points to remote system: ", err)
				}
			}
		case pts := <-chLocalEdgePoints:
			if connected {
				err = SendEdgePoints(up.ncRemote, pts.ID, pts.Parent, pts.Points, false)
				if err != nil {
					log.Println("Error sending edge points to remote system: ", err)
				}
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
					// we need to restart the sync connection
					up.disconnect()
					connectTimer.Reset(10 * time.Millisecond)
				case data.PointTypePeriod:
					checkPeriod()
					if connected {
						syncTicker.Reset(time.Duration(up.config.Period) *
							time.Second)
					}
				}
			}

			if up.config.SyncCountReset {
				up.config.SyncCount = 0
				up.config.SyncCountReset = false

				points := data.Points{
					{Type: data.PointTypeSyncCount, Value: 0},
					{Type: data.PointTypeSyncCountReset, Value: 0},
				}

				err = SendPoints(up.nc, SubjectNodePoints(up.config.ID), points, false)
				if err != nil {
					log.Println("Error resetting sync sync count: ", err)
				}
			}

		case pts := <-up.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &up.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
		case edge := <-up.chNewEdge:
			if !edge.local {
				// a new remote node was created, if it does not exist here,
				// create it

				// if parent is upstream root, then we don't worry about it
				if edge.parent == up.rootRemote.ID {
					break
				}

				nodes, err := GetNodes(up.ncLocal, edge.parent, edge.id, "", true)
				if err != nil {
					log.Println("Error getting local node: ", err)
					break
				}

				if len(nodes) > 0 {
					// local node already exists, so don't do anything
					break
				}
				// local node does not exist, so get the remote and send it
			fetchAgain:
				// edge points are sent first, so it may take a bit before we see
				// the node points
				time.Sleep(10 * time.Millisecond)
				nodes, err = GetNodes(up.ncRemote, edge.parent, edge.id, "", true)
				if err != nil {
					log.Println("Error getting node: ", err)
					break
				}
				for _, n := range nodes {
					// if type is not populated yet, try again
					if n.Type == "" {
						goto fetchAgain
					}
					err := up.sendNodesLocal(n)
					if err != nil {
						log.Println("Error chNewEdge sendNodesLocal: ", err)
					}
				}
			}

			err = up.subscribeRemoteNode(edge.parent, edge.id)
			if err != nil {
				log.Println("Error subscribing to new edge: ", err)
			}
		}
	}

	// clean up
	err = subLocalNodePoints.Unsubscribe()
	if err != nil {
		log.Println("Error unsubscribing node points from local bus: ", err)
	}

	err = subLocalEdgePoints.Unsubscribe()
	if err != nil {
		log.Println("Error unsubscribing edge points from local bus: ", err)
	}

	up.disconnect()
	up.ncLocal.Close()

	return nil
}

// Stop sends a signal to the Start function to exit
func (up *SyncClient) Stop(err error) {
	close(up.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (up *SyncClient) Points(nodeID string, points []data.Point) {
	up.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (up *SyncClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	up.newEdgePoints <- NewPoints{nodeID, parentID, points}
}

func (up *SyncClient) connect() error {
	if up.config.Disable {
		log.Printf("Sync %v disabled", up.config.Description)
		return nil
	}

	opts := EdgeOptions{
		URI:       up.config.URI,
		AuthToken: up.config.AuthToken,
		NoEcho:    true,
		Connected: func() {
			up.chConnected <- true
			log.Printf("Sync: %v: Remote Connected: %v\n",
				up.config.Description, up.config.URI)
		},
		Disconnected: func() {
			up.chConnected <- false
			log.Printf("Sync: %v: Remote Disconnected\n", up.config.Description)
		},
		Reconnected: func() {
			up.chConnected <- true
			log.Printf("Sync: %v: Remote Reconnected\n", up.config.Description)
		},
		Closed: func() {
			log.Printf("Sync: %v: Remote Closed\n", up.config.Description)
		},
	}

	var err error
	up.ncRemote, err = EdgeConnect(opts)

	if err != nil {
		return fmt.Errorf("Error connection to upstream NATS: %v", err)
	}

	return nil
}

func (up *SyncClient) subscribeRemoteNodePoints(id string) error {
	if _, ok := up.subRemoteNodePoints[id]; !ok {
		var err error
		up.subRemoteNodePoints[id], err = up.ncRemote.Subscribe(SubjectNodePoints(id), func(msg *nats.Msg) {
			nodeID, points, err := DecodeNodePointsMsg(msg)
			if err != nil {
				log.Println("Error decoding point: ", err)
				return
			}

			err = SendNodePoints(up.ncLocal, nodeID, points, false)
			if err != nil {
				log.Println("Error sending node points to remote system: ", err)
			}
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (up *SyncClient) subscribeRemoteEdgePoints(parent, id string) error {
	if _, ok := up.subRemoteEdgePoints[id]; !ok {
		var err error
		key := id + ":" + parent
		up.subRemoteEdgePoints[key], err = up.ncRemote.Subscribe(SubjectEdgePoints(id, parent),
			func(msg *nats.Msg) {
				nodeID, parentID, points, err := DecodeEdgePointsMsg(msg)
				if err != nil {
					log.Println("Error decoding point: ", err)
					return
				}

				err = SendEdgePoints(up.ncLocal, nodeID, parentID, points, false)
				if err != nil {
					log.Println("Error sending edge points to remote system: ", err)
				}
			})

		if err != nil {
			return err
		}
	}
	return nil
}

func (up *SyncClient) subscribeRemoteNode(parent, id string) error {
	err := up.subscribeRemoteNodePoints(id)
	if err != nil {
		return err
	}

	err = up.subscribeRemoteEdgePoints(parent, id)
	if err != nil {
		return err
	}

	// we walk through all local nodes and and subscribe to remote changes
	children, err := GetNodes(up.ncLocal, id, "all", "", true)
	if err != nil {
		return err
	}

	for _, c := range children {
		err := up.subscribeRemoteNode(c.Parent, c.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (up *SyncClient) disconnect() {
	for key, sub := range up.subRemoteNodePoints {
		err := sub.Unsubscribe()
		if err != nil {
			log.Println("Error unsubscribing from remote: ", err)
		}
		delete(up.subRemoteNodePoints, key)
	}

	for key, sub := range up.subRemoteEdgePoints {
		err := sub.Unsubscribe()
		if err != nil {
			log.Println("Error unsubscribing from remote: ", err)
		}
		delete(up.subRemoteEdgePoints, key)
	}

	up.initialSub = false
	if up.subRemoteUp != nil {
		err := up.subRemoteUp.Unsubscribe()
		if err != nil {
			log.Println("subRemoteUp.Unsubscribe() error: ", err)
		}
		up.subRemoteUp = nil
	}

	if up.ncRemote != nil {
		up.ncRemote.Close()
		up.ncRemote = nil
		up.rootRemote = data.NodeEdge{}
	}
}

// sendNodesRemote is used to send node and children over nats
// from one NATS server to another. Typically from the current instance
// to an upstream.
func (up *SyncClient) sendNodesRemote(node data.NodeEdge) error {
	if node.Parent == "root" {
		node.Parent = up.rootRemote.ID
	}

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
		err := up.sendNodesRemote(childNode)

		if err != nil {
			return fmt.Errorf("Error sending child node: %v", err)
		}
	}

	return nil
}

// sendNodesLocal is used to send node and children over nats
// from one NATS server to another. Typically from the current instance
// to an upstream.
func (up *SyncClient) sendNodesLocal(node data.NodeEdge) error {
	err := SendNode(up.ncLocal, node, up.config.ID)
	if err != nil {
		return err
	}

	// process child nodes
	childNodes, err := GetNodes(up.nc, node.ID, "all", "", false)
	if err != nil {
		return fmt.Errorf("Error getting node children: %v", err)
	}

	for _, childNode := range childNodes {
		err := up.sendNodesLocal(childNode)

		if err != nil {
			return fmt.Errorf("Error sending child node: %v", err)
		}
	}

	return nil
}

func (up *SyncClient) syncNode(parent, id string) error {
	var err error
	if up.rootRemote.ID == "" {
		up.rootRemote, err = GetRootNode(up.ncRemote)
		if err != nil {
			log.Printf("Sync: %v, error getting upstream root: %v\n", up.config.Description, err)
			return fmt.Errorf("Error getting upstream root: %v", err)
		}
	}

	if up.subRemoteUp == nil {
		subject := fmt.Sprintf("up.%v.*.*", up.rootLocal.ID)
		up.subRemoteUp, err = up.ncRemote.Subscribe(subject, func(msg *nats.Msg) {
			_, id, parent, points, err := DecodeUpEdgePointsMsg(msg)
			if err != nil {
				log.Println("Error decoding remote up points: ", err)
			} else {
				for _, p := range points {
					if p.Type == data.PointTypeTombstone &&
						p.Value == 0 {
						// we have a new node
						up.chNewEdge <- newEdge{
							parent: parent, id: id}
					}
				}
			}
		})

		if err != nil {
			log.Println("Error subscribing to remote up...: ", err)
		}
	}

	// Why do we do this?
	if parent == "root" {
		parent = "all"
	}

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

	nodeDeleted := false
	nodeFound := len(nodeUps) > 0

	if nodeFound {
		nodeDeleted = true
		for _, n := range nodeUps {
			ts, _ := n.IsTombstone()
			if !ts {
				nodeDeleted = false
				break
			}
		}
	}

	if nodeDeleted {
		nodeUp = nodeUps[0]
		// restore a node on the upstream
		// update the local tombstone timestamp so it is newer than the remote tombstone timestamp
		log.Printf("Sync: undeleting remote node: %v:%v\n", nodeUp.Parent, nodeUp.ID)
		pTS := data.Point{Time: time.Now(), Type: data.PointTypeTombstone, Value: 0}
		err := SendEdgePoint(up.ncRemote, nodeUp.ID, nodeUp.Parent, pTS, true)
		if err != nil {
			return fmt.Errorf("Error undeleting upstream node: %v", err)
		}

		// FIXME, remove this return
		return nil
	}

	if !nodeFound {
		log.Printf("Sync node %v does not exist, sending\n", nodeLocal.Desc())
		err := up.sendNodesRemote(nodeLocal)
		if err != nil {
			return fmt.Errorf("Error sending node upstream: %w", err)
		}

		err = up.subscribeRemoteNode(nodeLocal.ID, nodeLocal.Parent)
		if err != nil {
			return fmt.Errorf("Error subscribing to node changes: %w", err)
		}

		return nil
	}

	nodeUp = nodeUps[0]

	if nodeLocal.ID == up.rootLocal.ID {
		// we need to back out the edge points from the hash as don't want to sync those
		for _, p := range nodeUp.EdgePoints {
			nodeUp.Hash ^= p.CRC()
		}

		for _, p := range nodeLocal.EdgePoints {
			nodeLocal.Hash ^= p.CRC()
		}
	}

	if nodeUp.Hash == nodeLocal.Hash {
		// we're good!
		return nil
	}

	// only increment count once during sync
	if nodeLocal.ID == up.rootLocal.ID {
		up.config.SyncCount++
		points := data.Points{
			{Type: data.PointTypeSyncCount, Value: float64(up.config.SyncCount)},
		}

		err = SendPoints(up.nc, SubjectNodePoints(up.config.ID), points, false)
		if err != nil {
			log.Println("Error resetting sync sync count: ", err)
		}
	}

	log.Printf("sync %v: syncing node: %v, hash up: 0x%x, down: 0x%x ",
		up.config.Description,
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

	// only check edge points if we are not the root node
	if nodeLocal.ID != up.rootLocal.ID {
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
	}

	// sync child nodes
	children, err := GetNodes(up.ncLocal, nodeLocal.ID, "all", "", false)
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
					err := up.syncNode(nodeLocal.ID, child.ID)
					if err != nil {
						fmt.Println("Error syncing node: ", err)
					}
				}
			}
		}

		if !found {
			// need to send node upstream
			err := up.sendNodesRemote(child)
			if err != nil {
				log.Println("Error sending node upstream: ", err)
			}

			err = up.subscribeRemoteNode(child.Parent, child.ID)
			if err != nil {
				log.Println("Error subscribing to upstream: ", err)
			}
		}
	}

	for i, upChild := range upChildren {
		if _, ok := upChildProcessed[i]; !ok {
			err := up.sendNodesLocal(upChild)
			if err != nil {
				log.Println("Error getting node from upstream: ", err)
			}
			err = up.subscribeRemoteNode(upChild.Parent, upChild.ID)
			if err != nil {
				log.Println("Error subscribing to upstream: ", err)
			}
		}
	}

	return nil
}
