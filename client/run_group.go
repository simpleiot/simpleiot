package client

import (
	"sync"

	"github.com/oklog/run"
)

// RunGroup is used to group a list of clients and start/stop them
// currently a thin wrapper around run.Group that adds a Stop() function
type RunGroup struct {
	name     string
	stop     chan struct{}
	stopOnce sync.Once
	group    run.Group
}

// NewRunGroup creates a new client group
func NewRunGroup(name string) *RunGroup {
	return &RunGroup{name: name, stop: make(chan struct{})}
}

// Add client to group
func (g *RunGroup) Add(client RunStop) {
	g.group.Add(client.Run, client.Stop)
}

// Run clients. This function blocks until error or stopped.
// all clients must be added before runner is started
func (g *RunGroup) Run() error {
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
func (g *RunGroup) Stop(_ error) {
	g.stopOnce.Do(func() { close(g.stop) })
}
