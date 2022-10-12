package client

import (
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/oklog/run"
)

// BuiltInClients is used to manage the SIOT built in node clients
type BuiltInClients struct {
	nc       *nats.Conn
	stop     chan struct{}
	stopOnce sync.Once
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

	sc := NewManager(bic.nc, rootID, NewSerialDevClient)
	g.Add(sc.Start, sc.Stop)

	rc := NewManager(bic.nc, rootID, NewRuleClient)
	g.Add(rc.Start, rc.Stop)

	db := NewManager(bic.nc, rootID, NewDbClient)
	g.Add(db.Start, db.Stop)

	sg := NewManager(bic.nc, rootID, NewSignalGeneratorClient)
	g.Add(sg.Start, sg.Stop)

	g.Add(func() error {
		<-bic.stop
		return nil
	}, func(_ error) {
		bic.Stop(nil)
	})

	err = g.Run()

	return err
}

// Stop clients
func (bic *BuiltInClients) Stop(_ error) {
	bic.stopOnce.Do(func() { close(bic.stop) })
}
