package node

import (
	"log"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/nats"
)

// UpstreamManager looks for upstream nodes and creates new upstream connections
type UpstreamManager struct {
	nc         *natsgo.Conn
	upstreams  map[string]*Upstream
	rootNodeID string
}

// NewUpstreamManager is used to create a new upstream manager
func NewUpstreamManager(nc *natsgo.Conn, rootNodeID string) *UpstreamManager {
	return &UpstreamManager{
		nc:         nc,
		upstreams:  make(map[string]*Upstream),
		rootNodeID: rootNodeID,
	}
}

// Update queries DB for modbus nodes and synchronizes
// with internal structures and updates data
func (upm *UpstreamManager) Update() error {
	nodes, err := nats.GetNodeChildren(upm.nc, upm.rootNodeID, data.NodeTypeUpstream, false)
	if err != nil {
		return err
	}

	found := make(map[string]bool)

	for _, node := range nodes {
		found[node.ID] = true
		up, ok := upm.upstreams[node.ID]
		if !ok {
			var err error
			up, err = NewUpstream(upm.nc, node)
			if err != nil {
				log.Println("Error creating new Upstream: ", err)
				continue
			}
			upm.upstreams[node.ID] = up
		}
	}

	// remove upstreams that have been deleted
	for id, up := range upm.upstreams {
		_, ok := found[id]
		if !ok {
			// bus was deleted so close and clear it
			log.Println("removing upstream: ", up.nodeUp.Description)
			up.Stop()
			delete(upm.upstreams, id)
		}
	}

	return nil
}
