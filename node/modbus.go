package node

import (
	"errors"
	"fmt"
	"io"
	"log"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db/genji"
	"github.com/simpleiot/simpleiot/modbus"
	"github.com/simpleiot/simpleiot/nats"
)

// ModbusManager manages state of modbus
type ModbusManager struct {
	db     *genji.Db
	nc     *natsgo.Conn
	busses map[string]*Modbus
}

// NewModbusManager creates a new modbus manager
func NewModbusManager(db *genji.Db, nc *natsgo.Conn) *ModbusManager {
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
		return data.PointTypeErrorCount
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
	nodes, err := mm.db.NodeChildren(rootID, data.NodeTypeModbus)
	if err != nil {
		return err
	}

	found := make(map[string]bool)

	for _, node := range nodes {
		found[node.ID] = true
		bus, ok := mm.busses[node.ID]
		if !ok {
			var err error
			bus, err = NewModbus(mm.db, mm.nc, &node)
			if err != nil {
				log.Println("Error creating new modbus: ", err)
				continue
			}
			mm.busses[node.ID] = bus
		}

		err := bus.Check(&node)

		if err != nil {
			log.Println("Error initializing modbus port: ",
				node.ID, err)
			continue
		}

	}

	// remove busses that have been deleted
	for id, bus := range mm.busses {
		_, ok := found[id]
		if !ok {
			// bus was deleted so close and clear it
			log.Println("removing modbus on port: ", bus.node.portName)
			bus.Stop()
			delete(mm.busses, id)
		}
	}

	return nil
}

// ModbusNode is the node data from the database
type ModbusNode struct {
	nodeID             string
	busType            string
	id                 int // only used for server
	portName           string
	debugLevel         int
	baud               int
	pollPeriod         int
	errorCount         int
	errorCountCRC      int
	errorCountEOF      int
	errorCountReset    bool
	errorCountCRCReset bool
	errorCountEOFReset bool
}

// NewModbusNode converts a node to ModbusNode data structure
func NewModbusNode(node *data.NodeEdge) (*ModbusNode, error) {
	ret := ModbusNode{
		nodeID: node.ID,
	}

	var ok bool

	ret.busType, ok = node.Points.Text("", data.PointTypeClientServer, 0)
	if !ok {
		return nil, errors.New("Must define modbus client/server")
	}
	ret.portName, ok = node.Points.Text("", data.PointTypePort, 0)
	if !ok {
		return nil, errors.New("Must define modbus port name")
	}
	ret.baud, ok = node.Points.ValueInt("", data.PointTypeBaud, 0)
	if !ok {
		return nil, errors.New("Must define modbus baud")
	}

	ret.pollPeriod, ok = node.Points.ValueInt("", data.PointTypePollPeriod, 0)
	if !ok {
		return nil, errors.New("Must define modbus polling period")
	}

	ret.debugLevel, _ = node.Points.ValueInt("", data.PointTypeDebug, 0)
	ret.errorCount, _ = node.Points.ValueInt("", data.PointTypeErrorCount, 0)
	ret.errorCountCRC, _ = node.Points.ValueInt("", data.PointTypeErrorCountCRC, 0)
	ret.errorCountEOF, _ = node.Points.ValueInt("", data.PointTypeErrorCountEOF, 0)
	ret.errorCountReset, _ = node.Points.ValueBool("", data.PointTypeErrorCountReset, 0)
	ret.errorCountCRCReset, _ = node.Points.ValueBool("", data.PointTypeErrorCountCRCReset, 0)
	ret.errorCountEOFReset, _ = node.Points.ValueBool("", data.PointTypeErrorCountEOFReset, 0)

	if ret.busType == data.PointValueServer {
		var ok bool
		ret.id, ok = node.Points.ValueInt("", data.PointTypeID, 0)
		if !ok {
			return nil, errors.New("Must define modbus ID for server bus")
		}
	}

	return &ret, nil
}

// Modbus describes a modbus bus
type Modbus struct {
	// node data should only be changed through NATS, so that it is only changed in one place
	node *ModbusNode
	ios  map[string]*ModbusIO

	// data associated with running the bus
	db      *genji.Db
	nc      *natsgo.Conn
	runner  *ModbusRunner
	sub     *natsgo.Subscription
	chError <-chan error
}

// NewModbus creates a new bus from a node
func NewModbus(db *genji.Db, nc *natsgo.Conn, node *data.NodeEdge) (*Modbus, error) {
	bus := &Modbus{
		db:  db,
		nc:  nc,
		ios: make(map[string]*ModbusIO),
	}

	modbusNode, err := NewModbusNode(node)
	if err != nil {
		return nil, err
	}

	bus.node = modbusNode

	bus.sub, err = nc.Subscribe("node."+bus.node.nodeID+".points", func(msg *natsgo.Msg) {
		fmt.Println("msg: ", msg)
	})

	if err != nil {
		log.Println("Error subscribing to NATS topic: ", err)
	}

	return bus, nil
}

// Stop stops the bus and resets various fields
func (bus *Modbus) Stop() {
	if bus.chError != nil {
		if bus.runner != nil {
			bus.runner.Close()
		}
		bus.chError = nil
	}

	if bus.sub != nil {
		bus.sub.Unsubscribe()
	}
}

// Check verifies the Nodes for the bus and restarts it if anything
// has changed.
func (bus *Modbus) Check(node *data.NodeEdge) error {
	nodeBus, err := NewModbusNode(node)
	if err != nil {
		return err
	}

	err = bus.CheckIOs()
	if err != nil {
		return fmt.Errorf("Error checking modbus IOs: %w", err)
	}

	if nodeBus.errorCountReset {
		bus.Stop()
		p := data.Point{Type: data.PointTypeErrorCount, Value: 0}
		err := nats.SendPoint(bus.nc, bus.node.nodeID, p, true)
		if err != nil {
			log.Println("Send point error: ", err)
		}

		p = data.Point{Type: data.PointTypeErrorCountReset, Value: 0}
		err = nats.SendPoint(bus.nc, bus.node.nodeID, p, true)
		if err != nil {
			log.Println("Send point error: ", err)
		}
	}

	if nodeBus.errorCountCRCReset {
		bus.Stop()
		p := data.Point{Type: data.PointTypeErrorCountCRC, Value: 0}
		err := nats.SendPoint(bus.nc, bus.node.nodeID, p, true)
		if err != nil {
			log.Println("Send point error: ", err)
		}

		p = data.Point{Type: data.PointTypeErrorCountCRCReset, Value: 0}
		err = nats.SendPoint(bus.nc, bus.node.nodeID, p, true)
		if err != nil {
			log.Println("Send point error: ", err)
		}
	}

	if nodeBus.errorCountEOFReset {
		bus.Stop()
		p := data.Point{Type: data.PointTypeErrorCountEOF, Value: 0}
		err := nats.SendPoint(bus.nc, bus.node.nodeID, p, true)
		if err != nil {
			log.Println("Send point error: ", err)
		}

		p = data.Point{Type: data.PointTypeErrorCountEOFReset, Value: 0}
		err = nats.SendPoint(bus.nc, bus.node.nodeID, p, true)
		if err != nil {
			log.Println("Send point error: ", err)
		}
	}

	if nodeBus.busType != bus.node.busType ||
		nodeBus.portName != bus.node.portName ||
		nodeBus.baud != bus.node.baud ||
		nodeBus.id != bus.node.id ||
		nodeBus.debugLevel != bus.node.debugLevel ||
		nodeBus.pollPeriod != bus.node.pollPeriod {
		// bus has changed
		bus.Stop()
		bus.node = nodeBus
	}

	if bus.chError == nil {
		// need to start the bus. Need to pass a copy of data so that we don't run into concurrency
		// problems
		busNode := *bus.node
		ios := copyIos(bus.ios)
		runner := NewModbusRunner(bus.db, bus.nc, &busNode, ios)
		bus.chError = runner.Run()
	}

	return nil
}

// CheckIOs goes through ios on the bus and handles any config changes
func (bus *Modbus) CheckIOs() error {
	ioNodes, err := bus.db.NodeChildren(bus.node.nodeID, data.NodeTypeModbusIO)
	if err != nil {
		return err
	}

	found := make(map[string]bool)

	for _, ioNode := range ioNodes {
		found[ioNode.ID] = true
		io, ok := bus.ios[ioNode.ID]
		if !ok {
			// add ios
			var err error
			io, err = NewModbusIO(bus.node.busType, &ioNode)
			if err != nil {
				log.Println("Error creating new modbus: ", err)
				continue
			}
			bus.ios[ioNode.ID] = io
			bus.Stop()
		} else {
			// check if anything has changed
			newIO, err := NewModbusIO(bus.node.busType, &ioNode)
			if err != nil {
				log.Println("Error with modbus IO: ", err)
				continue
			}
			changed := io.Changed(newIO)
			if changed {
				bus.Stop()
				bus.ios[ioNode.ID] = newIO
			}
		}

		// check if error counters need reset
		if io.errorCountReset {
			bus.Stop()
			p := data.Point{Type: data.PointTypeErrorCount, Value: 0}
			err := nats.SendPoint(bus.nc, io.nodeID, p, true)
			if err != nil {
				log.Println("Error sending nats point: ", err)
			}

			p = data.Point{Type: data.PointTypeErrorCountReset, Value: 0}
			err = nats.SendPoint(bus.nc, io.nodeID, p, true)
			if err != nil {
				log.Println("Error sending nats point: ", err)
			}
		}

		if io.errorCountCRCReset {
			bus.Stop()
			p := data.Point{Type: data.PointTypeErrorCountCRC, Value: 0}
			err := nats.SendPoint(bus.nc, io.nodeID, p, true)
			if err != nil {
				log.Println("Error sending nats point: ", err)
			}

			p = data.Point{Type: data.PointTypeErrorCountCRCReset, Value: 0}
			err = nats.SendPoint(bus.nc, io.nodeID, p, true)
			if err != nil {
				log.Println("Error sending nats point: ", err)
			}
		}

		if io.errorCountEOFReset {
			bus.Stop()
			p := data.Point{Type: data.PointTypeErrorCountEOF, Value: 0}
			err := nats.SendPoint(bus.nc, io.nodeID, p, true)
			if err != nil {
				log.Println("Error sending nats point: ", err)
			}

			p = data.Point{Type: data.PointTypeErrorCountEOFReset, Value: 0}
			err = nats.SendPoint(bus.nc, io.nodeID, p, true)
			if err != nil {
				log.Println("Error sending nats point: ", err)
			}
		}
	}

	// remove ios that have been deleted
	for id, io := range bus.ios {
		_, ok := found[id]
		if !ok {
			// bus was deleted so close and clear it
			log.Println("modbus io removed: ", io.description)
			// FIXME, do we need to do anything here
			delete(bus.ios, id)
			bus.Stop()
		}
	}

	return nil
}

// ModbusIOMgr is used to manage modbus IOs
type ModbusIOMgr struct {
	IO  *ModbusIO
	Sub *natsgo.Subscription
}
