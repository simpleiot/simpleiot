package client

import (
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Client interface describes methods a Simple Iot client must implement.
// This is to be kept as simple as possible, and the ClientManager does all
// the heavy lifting of interacting with the rest of the SIOT system.
// Run should block until Stop is called.
// Run MUST return when Stop is called.
// Stop does not block -- wait until Run returns if you need to know the client
// is stopped.
// Points and EdgePoints are called when there are updates to the client node.
// The client Manager filters out all points with Origin set to "" because it
// assumes the point was generated by the client and the client already knows about it.
// Thus, if you want points to get to a client, Origin must be set.
type Client interface {
	RunStop

	Points(string, []data.Point)
	EdgePoints(string, string, []data.Point)
}

// DefaultClients returns an actor for the default group of built in clients
func DefaultClients(nc *nats.Conn) (*Group, error) {
	g := NewGroup("Default clients")

	sc := NewManager(nc, NewSerialDevClient, nil)
	g.Add(sc)

	cb := NewManager(nc, NewCanBusClient, nil)
	g.Add(cb)

	rc := NewManager(nc, NewRuleClient, nil)
	g.Add(rc)

	db := NewManager(nc, NewDbClient, nil)
	g.Add(db)

	sg := NewManager(nc, NewSignalGeneratorClient, nil)
	g.Add(sg)

	sync := NewManager(nc, NewSyncClient, nil)
	g.Add(sync)

	metrics := NewManager(nc, NewMetricsClient, nil)
	g.Add(metrics)

	particle := NewManager(nc, NewParticleClient, nil)
	g.Add(particle)

	shelly := NewManager(nc, NewShellyClient, nil)
	g.Add(shelly)

	shellyIO := NewManager(nc, NewShellyIOClient, []string{data.NodeTypeShelly})
	g.Add(shellyIO)

	ntp := NewManager(nc, NewNTPClient, nil)
	g.Add(ntp)

	zm := NewManager(nc, NewZMiniClient)
	g.Add(zm)

	return g, nil
}
