package client

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

func mapKey(node data.NodeEdge) string {
	return node.Parent + node.ID
}

type clientState[T any] struct {
	nc           *nats.Conn
	node         data.NodeEdge
	nec          data.NodeEdgeChildren
	construct    func(*nats.Conn, T) Client
	client       Client
	stopPointSub func()
	stopEdgeSub  func()
}

func newClientState[T any](nc *nats.Conn, construct func(*nats.Conn, T) Client,
	n data.NodeEdge) *clientState[T] {

	ret := &clientState[T]{
		node:      n,
		nc:        nc,
		construct: construct,
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

	err = cs.sub(cs.client, cs.node.ID, cs.node.Parent, mapKey(cs.node))
	if err != nil {

	}

	return cs.client.Start()
}

func (cs *clientState[T]) sub(client Client, nodeID, parentID, key string) error {
	stopNodeSub, err := SubscribePoints(cs.nc, nodeID, func(points []data.Point) {
		client.Points(points)
	})
	if err != nil {
		return fmt.Errorf("client manager sub error: %v", err)
	}

	cs.stopPointSub = stopNodeSub

	stopEdgeSub, err := SubscribeEdgePoints(cs.nc, nodeID, parentID, func(points []data.Point) {
		client.EdgePoints(points)
		for _, p := range points {
			if p.Type == data.PointTypeTombstone {
				cs.stop(nil)
			}
		}
	})
	if err != nil {
		return fmt.Errorf("client manager edge sub error: %v", err)
	}
	cs.stopEdgeSub = stopEdgeSub

	return nil
}

func (cs *clientState[T]) stop(err error) {
	if cs.stopPointSub != nil {
		cs.stopPointSub()
	}

	if cs.stopEdgeSub != nil {
		cs.stopEdgeSub()
	}

	cs.client.Stop(err)
}
