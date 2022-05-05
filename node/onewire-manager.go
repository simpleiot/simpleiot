package node

import (
	"log"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/nats"
)

type oneWireManager struct {
	nc         *natsgo.Conn
	busses     map[string]*oneWire
	rootNodeID string
}

func newOneWireManager(nc *natsgo.Conn, rootNodeID string) *oneWireManager {
	return &oneWireManager{
		nc:         nc,
		busses:     make(map[string]*oneWire),
		rootNodeID: rootNodeID,
	}
}

func (owm *oneWireManager) update() error {
	nodes, err := nats.GetNodeChildren(owm.nc, owm.rootNodeID, data.NodeTypeOneWire, false, false)
	if err != nil {
		return err
	}

	found := make(map[string]bool)

	for _, node := range nodes {
		found[node.ID] = true
		bus, ok := owm.busses[node.ID]
		if !ok {
			var err error
			bus, err = newOneWire(owm.nc, node)
			if err != nil {
				log.Println("Error creating new modbus: ", err)
				continue
			}
			owm.busses[node.ID] = bus
		}
	}

	// remove busses that have been deleted
	for id, bus := range owm.busses {
		_, ok := found[id]
		if !ok {
			// bus was deleted so close and clear it
			log.Println("removing onewire bus")
			bus.Stop()
			delete(owm.busses, id)
		}
	}

	return nil

}
