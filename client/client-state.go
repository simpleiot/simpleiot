package client

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

func mapKey(node data.NodeEdge) string {
	return node.Parent + "-" + node.ID
}

type clientState[T any] struct {
	nc        *nats.Conn
	node      data.NodeEdge
	nec       data.NodeEdgeChildren
	construct func(*nats.Conn, T) Client

	// subscription to listen for new points
	upSub  *nats.Subscription
	client Client

	stopOnce sync.Once
	chStop   chan struct{}
}

func newClientState[T any](nc *nats.Conn, construct func(*nats.Conn, T) Client,
	n data.NodeEdge) *clientState[T] {

	ret := &clientState[T]{
		node:      n,
		nc:        nc,
		construct: construct,
		chStop:    make(chan struct{}),
	}

	return ret
}

func (cs *clientState[T]) start() (err error) {
	c, err := GetNodeChildren(cs.nc, cs.node.ID, "", false, false)
	if err != nil {
		err = fmt.Errorf("Error getting children: %v", err)
		return
	}

	ncc := make([]data.NodeEdgeChildren, len(c))

	for i, nci := range c {
		ncc[i] = data.NodeEdgeChildren{NodeEdge: nci, Children: nil}
	}

	cs.nec = data.NodeEdgeChildren{NodeEdge: cs.node, Children: ncc}

	var config T

	err = data.Decode(cs.nec, &config)
	if err != nil {
		err = fmt.Errorf("Error decoding node: %v", err)
		return
	}

	cs.client = cs.construct(cs.nc, config)

	// Set up subscriptions
	subject := fmt.Sprintf("up.%v.>", cs.node.ID)

	cs.upSub, err = cs.nc.Subscribe(subject, func(msg *nats.Msg) {
		points, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points")
			return
		}
		for _, p := range points {
			if p.Origin == "" {
				// point came from the owning node, we already know about it
				return
			}
		}

		// find node ID for points
		chunks := strings.Split(msg.Subject, ".")
		if len(chunks) == 4 {
			// node points
			for _, p := range points {
				if p.Type == data.PointTypeNodeType {
					cs.stop(nil)
					return
				}
			}

			// send node points to client
			cs.client.Points(chunks[2], points)

		} else if len(chunks) == 5 {
			// edge points
			for _, p := range points {
				if p.Type == data.PointTypeTombstone {
					// a node was deleted, stop client and restart
					cs.stop(nil)
					return
				}
			}

			// send edge points to client
			cs.client.EdgePoints(chunks[2], chunks[3], points)
		} else {
			log.Println("up subject malformed: ", msg.Subject)
			return
		}

	})

	if err != nil {
		return
	}

	chClientStopped := make(chan struct{})

	go func() {
		// the following blocks until client exits
		err := cs.client.Start()
		if err != nil {
			log.Printf("Client Start %v %v returned error: %v\n",
				cs.node.Type, cs.node.ID, err)
		}
		close(chClientStopped)
	}()

	<-cs.chStop
	cs.upSub.Unsubscribe()
	cs.client.Stop(nil)

	select {
	case <-chClientStopped:
		// everything is OK
	case <-time.After(5 * time.Second):
		log.Println("Timeout stopping client: ", cs.node.Type, cs.node.ID)
	}

	return nil
}

func (cs *clientState[T]) stop(err error) {
	cs.stopOnce.Do(func() { close(cs.chStop) })
}
