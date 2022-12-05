package client

import (
	"sync"

	"github.com/oklog/run"
)

// Group is used to group a list of clients and start/stop them
// currently a thin wrapper around run.Group that adds a Stop() function
type Group struct {
	name     string
	stop     chan struct{}
	stopOnce sync.Once
	group    run.Group
}

// NewGroup creates a new client group
func NewGroup(name string) *Group {
	return &Group{name: name, stop: make(chan struct{})}
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
	}, func(err error) {
		g.Stop(nil)
	})

	err := g.group.Run()

	return err
}

// Stop clients
func (g *Group) Stop(err error) {
	g.stopOnce.Do(func() { close(g.stop) })
}
