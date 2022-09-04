package client

import (
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/oklog/run"
)

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

	nodes, err := GetNode(bic.nc, "root", "")
	if err != nil {
		return fmt.Errorf("Error starting build in clients getting root node: %v", err)
	}

	if len(nodes) < 1 {
		return fmt.Errorf("Error starting build in clients no root node")
	}

	rootID = nodes[0].ID
	_ = rootID

	/*
		sc := NewManager(bic.nc, rootID, NewSerialDevClient)
		g.Add(sc.Start, sc.Stop)

		rc := NewManager(bic.nc, rootID, NewRuleClient)
		g.Add(rc.Start, rc.Stop)
	*/

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
