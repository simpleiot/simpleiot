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

// clientState wraps the client, passes in initial state, and then runs the client
type clientState[T any] struct {
	nc   *nats.Conn
	node data.NodeEdge
	nec  data.NodeEdgeChildren

	client Client

	// crashes is how many times in a row this client panicked before this
	// run. The manager sets it so a restart can back off.
	crashes int

	stopOnce sync.Once
	chStop   chan struct{}

	// crashErr is the panic that stopped the client, in Run or in a
	// points callback, so run can report it to the manager
	crashMu  sync.Mutex
	crashErr error
}

func newClientState[T any](nc *nats.Conn, construct func(*nats.Conn, T) Client,
	n data.NodeEdge) (*clientState[T], error) {

	c, err := GetNodes(nc, n.ID, "all", "", false)
	if err != nil {
		return nil, fmt.Errorf("error getting children: %v", err)
	}

	ncc := make([]data.NodeEdgeChildren, len(c))

	for i, nci := range c {
		ncc[i] = data.NodeEdgeChildren{NodeEdge: nci, Children: nil}
	}

	nec := data.NodeEdgeChildren{NodeEdge: n, Children: ncc}

	var config T

	err = data.Decode(nec, &config)
	if err != nil {
		return nil, fmt.Errorf("error decoding node: %w", err)
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

// what names the client in log lines
func (cs *clientState[T]) what() string {
	return fmt.Sprintf("client %v %v", cs.node.Type, cs.node.ID)
}

// run blocks until the client stops. It returns the *panicError when the
// client stopped because it panicked, so the manager can record it and back
// off before starting the client again.
func (cs *clientState[T]) run() error {

	chClientStopped := make(chan struct{})

	go func() {
		// the following blocks until client exits
		err := runRecovered(cs.what(), cs.client.Run)

		var pe *panicError
		if errors.As(err, &pe) {
			cs.crash(pe)
		} else if err != nil {
			log.Printf("%v Run returned error: %v\n", cs.what(), err)
		}
		close(chClientStopped)
	}()

	<-cs.chStop
	cs.client.Stop(nil)

	select {
	case <-chClientStopped:
		// everything is OK
	case <-time.After(5 * time.Second):
		log.Println("Timeout stopping", cs.what())
	}

	cs.crashMu.Lock()
	defer cs.crashMu.Unlock()

	return cs.crashErr
}

// crash records that the client panicked and stops it, so run returns and the
// manager starts the client again
func (cs *clientState[T]) crash(err error) {
	cs.crashMu.Lock()
	if cs.crashErr == nil {
		cs.crashErr = err
	}
	cs.crashMu.Unlock()

	cs.stop(nil)
}

func (cs *clientState[T]) stop(_ error) {
	cs.stopOnce.Do(func() { close(cs.chStop) })
}
