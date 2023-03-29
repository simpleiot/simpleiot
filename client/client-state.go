package client

import (
	"fmt"
	"log"
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

func (cs *clientState[T]) run() (err error) {
	c, err := GetNodes(cs.nc, cs.node.ID, "all", "", false)
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

	chClientStopped := make(chan struct{})

	go func() {
		// the following blocks until client exits
		err := cs.client.Run()
		if err != nil {
			log.Printf("Client Run %v %v returned error: %v\n",
				cs.node.Type, cs.node.ID, err)
		}
		close(chClientStopped)
	}()

	<-cs.chStop
	cs.client.Stop(nil)

	select {
	case <-chClientStopped:
		// everything is OK
	case <-time.After(5 * time.Second):
		log.Println("Timeout stopping client: ", cs.node.Type, cs.node.ID)
	}

	return nil
}

func (cs *clientState[T]) stop(_ error) {
	cs.stopOnce.Do(func() { close(cs.chStop) })
}
