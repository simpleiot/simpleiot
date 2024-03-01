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
func (g *Group) Add(client RunStop) {
	g.group.Add(client.Run, client.Stop)
}

// Run clients. This function blocks until error or stopped.
// all clients must be added before runner is started
func (g *Group) Run() error {
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
func (g *Group) Stop(_ error) {
	g.stopOnce.Do(func() { close(g.stop) })
}
