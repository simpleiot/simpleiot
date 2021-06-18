package node

import (
	"io"
	"log"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/modbus"
	"github.com/simpleiot/simpleiot/nats"
)

// ModbusManager manages state of modbus
type ModbusManager struct {
	nc         *natsgo.Conn
	busses     map[string]*Modbus
	rootNodeID string
}

// NewModbusManager creates a new modbus manager
func NewModbusManager(nc *natsgo.Conn, rootNodeID string) *ModbusManager {
	return &ModbusManager{
		nc:         nc,
		busses:     make(map[string]*Modbus),
		rootNodeID: rootNodeID,
	}
}

func modbusErrorToPointType(err error) string {
	switch err {
	case io.EOF:
		return data.PointTypeErrorCountEOF
	case modbus.ErrCRC:
		return data.PointTypeErrorCountCRC
	default:
		return ""
	}
}

func copyIos(in map[string]*ModbusIO) map[string]*ModbusIO {
	out := make(map[string]*ModbusIO)
	for k, v := range in {
		io := *v
		out[k] = &io
	}
	return out
}

// Update queries DB for modbus nodes and synchronizes
// with internal structures and updates data
func (mm *ModbusManager) Update() error {
	nodes, err := nats.GetNodeChildren(mm.nc, mm.rootNodeID, data.NodeTypeModbus, false)
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
