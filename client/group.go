package client

import (
	"sync"

	"github.com/oklog/run"
)

// Group is used to group a list of clients and start/stop them
// currently a thin wrapper around run.Group
type Group struct {
	stop     chan struct{}
	stopOnce sync.Once
	group    run.Group
}

// NewGroup creates a new client group
func NewGroup() *Group {
	return &Group{stop: make(chan struct{})}
}

// Add client to group
func (g *Group) Add(client StartStop) {
	g.group.Add(client.Start, client.Stop)
}

// Start clients. This function blocks until error or stopped.
// all clients must be added before runner is started
func (g *Group) Start() error {
	g.group.Add(func() error {
		<-g.stop
		return nil
	}, func(_ error) {
		g.Stop(nil)
	})

	err := g.group.Run()

	return err
}

// Stop clients
func (g *Group) Stop(err error) {
	g.stopOnce.Do(func() { close(g.stop) })
}
