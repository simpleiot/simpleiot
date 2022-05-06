package node

import (
	"log"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/nats"
)

type oneWire struct {
	node   data.NodeEdge
	owNode *oneWireNode
	ios    map[string]*oneWireIO

	// data associated with running the bus
	nc  *natsgo.Conn
	sub *natsgo.Subscription

	chDone  chan bool
	chPoint chan pointWID
}

func newOneWire(nc *natsgo.Conn, node data.NodeEdge) (*oneWire, error) {
	bus := &oneWire{
		nc:      nc,
		node:    node,
		ios:     make(map[string]*oneWireIO),
		chDone:  make(chan bool),
		chPoint: make(chan pointWID),
	}

	oneWireNode, err := newOneWireNode(node)
	if err != nil {
		return nil, err
	}

	bus.owNode = oneWireNode

	// closure is required so we don't get races accessing bus.busNode
	func(id string) {
		bus.sub, err = nc.Subscribe("node."+bus.owNode.nodeID+".points", func(msg *natsgo.Msg) {
			points, err := data.PbDecodePoints(msg.Data)
			if err != nil {
				// FIXME, send over channel
				log.Println("Error decoding node data: ", err)
				return
			}

			for _, p := range points {
				bus.chPoint <- pointWID{id, p}
			}
		})
	}(bus.owNode.nodeID)

	if err != nil {
		return nil, err
	}

	go bus.Run()

	return bus, nil
}

// Stop stops the bus and resets various fields
func (ow *oneWire) Stop() {
	if ow.sub != nil {
		err := ow.sub.Unsubscribe()
		if err != nil {
			log.Println("Error unsubscribing from bus: ", err)
		}
	}
	for _, io := range ow.ios {
		io.Stop()
	}
	ow.chDone <- true
}

// CheckIOs goes through ios on the bus and handles any config changes
func (ow *oneWire) CheckIOs() error {
	nodes, err := nats.GetNodeChildren(ow.nc, ow.owNode.nodeID, data.NodeTypeModbusIO, false, false)
	if err != nil {
		return err
	}

	found := make(map[string]bool)

	for _, node := range nodes {
		found[node.ID] = true
		io, ok := ow.ios[node.ID]
		if !ok {
			// add ios
			var err error
			ioNode, err := newOneWireIONode(&node)
			if err != nil {
				log.Println("Error with IO node: ", err)
				continue
			}
			io, err = newOneWireIO(ow.nc, ioNode, ow.chPoint)
			if err != nil {
				log.Println("Error creating new modbus IO: ", err)
				continue
			}
			ow.ios[node.ID] = io
		}
	}

	// remove ios that have been deleted
	for id, io := range ow.ios {
		_, ok := found[id]
		if !ok {
			// io was deleted so close and clear it
			log.Println("modbus io removed: ", io.ioNode.description)
			io.Stop()
			delete(ow.ios, id)
		}
	}

	return nil
}

func (ow *oneWire) Run() {
	for {
		select {
		case <-ow.chDone:
			return
		}
	}
}
