package client

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

func mapKey(node data.NodeEdge) string {
	return node.Parent + node.ID
}

type clientState[T any] struct {
	nc        *nats.Conn
	node      data.NodeEdge
	nec       data.NodeEdgeChildren
	construct func(*nats.Conn, T) Client
	client    Client

	// the following maps must be locked before access
	lock         sync.Mutex
	stopEdgeSubs map[string]func()

	// subscription to listen for new points
	upSub *nats.Subscription
}

func newClientState[T any](nc *nats.Conn, construct func(*nats.Conn, T) Client,
	n data.NodeEdge) *clientState[T] {

	ret := &clientState[T]{
		node:         n,
		nc:           nc,
		construct:    construct,
		stopEdgeSubs: make(map[string]func()),
	}

	return ret
}

func (cs *clientState[T]) start() error {
	c, err := GetNodeChildren(cs.nc, cs.node.ID, "", false, false)
	if err != nil {
		return fmt.Errorf("Error getting children: %v", err)
	}

	ncc := make([]data.NodeEdgeChildren, len(c))

	for i, nci := range c {
		ncc[i] = data.NodeEdgeChildren{NodeEdge: nci, Children: nil}
	}

	cs.nec = data.NodeEdgeChildren{NodeEdge: cs.node, Children: ncc}

	var config T

	err = data.Decode(cs.nec, &config)
	if err != nil {
		return fmt.Errorf("Error decoding node: %v", err)
	}

	cs.client = cs.construct(cs.nc, config)

	// Set up subscriptions
	cs.lock.Lock()

	subject := fmt.Sprintf("up.%v.>", cs.node.ID)

	cs.upSub, err = cs.nc.Subscribe(subject, func(msg *nats.Msg) {
		points, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points")
			return
		}

		// find node ID for points
		chunks := strings.Split(msg.Subject, ".")
		if len(chunks) < 4 {
			log.Println("up subject malformed: ", msg.Subject)
			return
		}

		cs.client.Points(chunks[2], points)

		for _, p := range points {
			if p.Type == data.PointTypeNodeType {
				// restart this client as there is a new node
				cs.stop(nil)
			}
		}
	})

	if err != nil {
		cs.lock.Unlock()
		return err
	}

	subEdge := func(nc *nats.Conn, node data.NodeEdge) (func(), error) {
		return SubscribeEdgePoints(cs.nc, node.ID, node.Parent, func(points []data.Point) {
			cs.client.EdgePoints(node.ID, node.Parent, points)
			for _, p := range points {
				if p.Type == data.PointTypeTombstone && p.Value == 1 {
					// a node was deleted, stop client and restart
					cs.stop(nil)
				}
			}
		})
	}

	cs.stopEdgeSubs[mapKey(cs.node)], err = subEdge(cs.nc, cs.node)
	if err != nil {
		cs.lock.Unlock()
		return fmt.Errorf("edge sub error: %v", err)
	}

	for _, n := range cs.nec.Children {
		ne := n.NodeEdge
		cs.stopEdgeSubs[mapKey(ne)], err = subEdge(cs.nc, ne)
		if err != nil {
			cs.lock.Unlock()
			return fmt.Errorf("edge sub error: %v", err)
		}
	}

	cs.lock.Unlock()

	// the following blocks until client exits
	return cs.client.Start()
}

func (cs *clientState[T]) stop(err error) {
	cs.lock.Lock()
	for _, f := range cs.stopEdgeSubs {
		if f != nil {
			f()
		}
	}
	cs.lock.Unlock()

	cs.client.Stop(err)
}
