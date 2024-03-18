package client

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// clientState wraps the client, passes in initial state, and then runs the client
type clientState[T any] struct {
	nc   *nats.Conn
	node data.NodeEdge
	nec  data.NodeEdgeChildren

	client Client

	stopOnce sync.Once
	chStop   chan struct{}
}

func newClientState[T any](nc *nats.Conn, construct func(*nats.Conn, T) Client,
	n data.NodeEdge) (*clientState[T], error) {

	c, err := GetNodes(nc, n.ID, "all", "", false)
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
		return nil, fmt.Errorf("Error decoding node: %w", err)
	}

	client := construct(nc, config)

	ret := &clientState[T]{
		nc:     nc,
		node:   n,
		nec:    nec,
		client: client,
		chStop: make(chan struct{}),
	}

	return ret, nil
}

func (cs *clientState[T]) run() (err error) {

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
		log.Println("Timeout stopping client:", cs.node.Type, cs.node.ID)
	}

	return nil
}

func (cs *clientState[T]) stop(_ error) {
	cs.stopOnce.Do(func() { close(cs.chStop) })
}
