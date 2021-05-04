package node

import (
	"log"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
)

// UpstreamManager looks for upstream nodes and creates new upstream connections
type UpstreamManager struct {
	db        *db.Db
	nc        *natsgo.Conn
	upstreams map[string]*Upstream
}

// NewUpstreamManager is used to create a new upstream manager
func NewUpstreamManager(db *db.Db, nc *natsgo.Conn) *UpstreamManager {
	return &UpstreamManager{
		nc:        nc,
		db:        db,
		upstreams: make(map[string]*Upstream),
	}
}

// Update queries DB for modbus nodes and synchronizes
// with internal structures and updates data
func (upm *UpstreamManager) Update() error {
	rootID := upm.db.RootNodeID()
	nodes, err := upm.db.NodeDescendents(rootID, data.NodeTypeUpstream, false)
	if err != nil {
		return err
	}

	found := make(map[string]bool)

	for _, node := range nodes {
		found[node.ID] = true
		up, ok := upm.upstreams[node.ID]
		if !ok {
			var err error
			up, err = NewUpstream(upm.db, upm.nc, node)
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
