package client

import (
	"errors"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/oklog/run"
	"github.com/simpleiot/simpleiot/data"
)

// Client interface describes methods a Simple Iot client must implement.
// This is to be kept as simple as possible, and the ClientManager does all
// the heavy lifting of interacting with the rest of the SIOT system.
// Start should block until Stop is called.
// Start MUST return when Stop is called.
// Stop does not block -- wait until Start returns if you need to know the client
// is stopped.
// Points and EdgePoints are called when there are updates to the client node.
type Client interface {
	Start() error
	Stop(error)
	Points([]data.Point)
	EdgePoints([]data.Point)
}

// BuiltInClients is used to manage the SIOT built in node clients
type BuiltInClients struct {
	nc   *nats.Conn
	stop chan struct{}
}

// NewBuiltInClients creates a new built in client manager
func NewBuiltInClients(nc *nats.Conn) *BuiltInClients {
	return &BuiltInClients{
		nc:   nc,
		stop: make(chan struct{}),
	}
}

// Start clients. This function blocks until error or stopped.
func (bic *BuiltInClients) Start() error {
	var g run.Group
	var rootID string

	// get root ID
gotId:
	for {
		select {
		case <-time.After(time.Second):
			nodes, err := GetNode(bic.nc, "root", "")
			if err != nil {
				continue
			}
			if len(nodes) < 1 {
				continue
			}
			rootID = nodes[0].ID
			break gotId

		case <-bic.stop:
			return nil
		}
	}

	sc := NewManager(bic.nc, rootID, newSerialDevClient)

	g.Add(sc.Start, sc.Stop)

	// provide actor to close run group
	stopStop := make(chan struct{})

	g.Add(func() error {
		select {
		case <-bic.stop:
			return errors.New("SIOT Built-in clients stopped")
		case <-stopStop:
			return nil
		}
	}, func(_ error) {
		close(stopStop)
	})

	stopped := make(chan error)

	go func() {
		stopped <- g.Run()
	}()

	for {
		select {
		case <-bic.stop:

		case err := <-stopped:
			return err
		}
	}
}

// Stop clients
func (bic *BuiltInClients) Stop(_ error) {
	close(bic.stop)
}
