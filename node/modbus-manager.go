package node

import (
	"log"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// ModbusManager manages state of modbus
type ModbusManager struct {
	nc         *nats.Conn
	busses     map[string]*Modbus
	rootNodeID string
}

// NewModbusManager creates a new modbus manager
func NewModbusManager(nc *nats.Conn, rootNodeID string) *ModbusManager {
	return &ModbusManager{
		nc:         nc,
		busses:     make(map[string]*Modbus),
		rootNodeID: rootNodeID,
	}
}

// Update queries DB for modbus nodes and synchronizes
// with internal structures and updates data
func (mm *ModbusManager) Update() error {
	nodes, err := client.GetNodeChildren(mm.nc, mm.rootNodeID, data.NodeTypeModbus, false, false)
	if err != nil {
		return err
	}

	found := make(map[string]bool)

	for _, node := range nodes {
		found[node.ID] = true
		bus, ok := mm.busses[node.ID]
		if !ok {
			var err error
			bus, err = NewModbus(mm.nc, node)
			if err != nil {
				log.Println("Error creating new modbus: ", err)
				continue
			}
			mm.busses[node.ID] = bus
		}
	}

	// remove busses that have been deleted
	for id, bus := range mm.busses {
		_, ok := found[id]
		if !ok {
			// bus was deleted so close and clear it
			log.Println("removing modbus on port: ", bus.busNode.portName)
			bus.Stop()
			delete(mm.busses, id)
		}
	}

	return nil
}
