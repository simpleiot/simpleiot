package node

import (
	"io"
	"log"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
	"github.com/simpleiot/simpleiot/modbus"
)

// ModbusManager manages state of modbus
type ModbusManager struct {
	db     *db.Db
	nc     *natsgo.Conn
	busses map[string]*Modbus
}

// NewModbusManager creates a new modbus manager
func NewModbusManager(db *db.Db, nc *natsgo.Conn) *ModbusManager {
	return &ModbusManager{
		db:     db,
		nc:     nc,
		busses: make(map[string]*Modbus),
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
	rootID := mm.db.RootNodeID()
	// TODO this should eventually be modified to not recurse into
	// child devices
	nodes, err := mm.db.NodeDescendents(rootID, data.NodeTypeModbus, true)
	if err != nil {
		return err
	}

	found := make(map[string]bool)

	for _, node := range nodes {
		found[node.ID] = true
		bus, ok := mm.busses[node.ID]
		if !ok {
			var err error
			bus, err = NewModbus(mm.db, mm.nc, node)
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
