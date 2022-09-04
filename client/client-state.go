package client

import (
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

func mapKey(node data.NodeEdge) string {
	return node.Parent + node.ID
}

type clientState[T any] struct {
	nc           *nats.Conn
	node         data.NodeEdgeChildren
	client       Client
	stopPointSub func()
	stopEdgeSub  func()
	chStop       chan struct{}
}

func newClientState[T any](nc *nats.Conn, construct func(*nats.Conn, T) Client,
	n data.NodeEdge, stopped func(error)) (*clientState[T], error) {

	ret := &clientState[T]{
		nc:     nc,
		chStop: make(chan struct{}),
	}

	c, err := GetNodeChildren(nc, n.ID, "", false, false)
	if err != nil {
		return nil, fmt.Errorf("Error getting children: %v", err)
	}

	ncc := make([]data.NodeEdgeChildren, len(c))

	for i, nci := range c {
		ncc[i] = data.NodeEdgeChildren{NodeEdge: nci, Children: nil}
	}

	nec := data.NodeEdgeChildren{NodeEdge: n, Children: ncc}

	var config T

	err = data.Decode(nec, &config)
	if err != nil {
		return nil, fmt.Errorf("Error decoding node: %v", err)
	}

	client := construct(nc, config)
	ret.client = client

	err = ret.sub(client, n.ID, n.Parent, mapKey(n))
	if err != nil {

	}

	// start client for new node
	go func(client Client) {
		err := client.Start()
		if err != nil {
			log.Println("Node client returned error: ", err)
		}

		stopped(err)
	}(client)

	return ret, nil
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
