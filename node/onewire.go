package node

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

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

	go bus.run()

	return bus, nil
}

// stop stops the bus and resets various fields
func (ow *oneWire) stop() {
	if ow.sub != nil {
		err := ow.sub.Unsubscribe()
		if err != nil {
			log.Println("Error unsubscribing from bus: ", err)
		}
	}
	for _, io := range ow.ios {
		io.stop()
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
			io.stop()
			delete(ow.ios, id)
		}
	}

	return nil
}

// checkIOs goes through ios on the bus and handles any config changes
func (ow *oneWire) checkIOs() error {
	nodes, err := nats.GetNodeChildren(ow.nc, ow.owNode.nodeID, data.NodeTypeOneWireIO, false, false)
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
			io.stop()
			delete(ow.ios, id)
		}
	}

	return nil
}

func (ow *oneWire) detect() {
	// detect one wire busses
	dirs, _ := filepath.Glob("/sys/bus/w1/devices/28-*")

	for _, dir := range dirs {
		f, _ := os.Stat(dir)
		if f.IsDir() {
			id := path.Base(dir)
			found := false
			for _, io := range ow.ios {
				if io.ioNode.id == id {
					found = true
					break
				}
			}

			if !found {
				log.Println("adding 1-wire IO: ", id)

				n := data.NodeEdge{
					Type:   data.NodeTypeOneWireIO,
					Parent: ow.owNode.nodeID,
					Points: data.Points{
						data.Point{
							Type: data.PointTypeID,
							Text: id,
						},
						data.Point{
							Type: data.PointTypeDescription,
							Text: "New IO, please edit",
						},
					},
				}

				err := nats.SendNode(ow.nc, n)
				if err != nil {
					log.Println("Error sending new 1-wire IO: ", err)
				}
			}
		}
	}
}

func (ow *oneWire) run() {
	// if we reset any error count, we set this to avoid continually resetting
	scanTimer := time.NewTicker(24 * time.Hour)

	setScanTimer := func() {
		pollPeriod := ow.owNode.pollPeriod
		if pollPeriod <= 0 {
			pollPeriod = 3000
		}
		scanTimer.Reset(time.Millisecond * time.Duration(pollPeriod))
	}

	setScanTimer()

	for {
		select {
		case point := <-ow.chPoint:
			p := point.point
			if point.id == ow.owNode.nodeID {
				ow.node.AddPoint(p)
				var err error
				ow.owNode, err = newOneWireNode(ow.node)
				if err != nil {
					log.Println("Error updating OW node: ", err)
				}

				switch point.point.Type {
				case data.PointTypePollPeriod:
					setScanTimer()
				case data.PointTypeErrorCountReset:
					if ow.owNode.errorCountReset {
						p := data.Point{Type: data.PointTypeErrorCount, Value: 0}
						err := nats.SendNodePoint(ow.nc, ow.owNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}

						p = data.Point{Type: data.PointTypeErrorCountReset, Value: 0}
						err = nats.SendNodePoint(ow.nc, ow.owNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}
					}
				}
				continue
			}

			io, ok := ow.ios[point.id]
			if !ok {
				log.Println("1-wire received point for unknown node: ", point.id)
				continue
			}

			err := io.point(p)
			if err != nil {
				log.Println("Error updating node point")
			}

		case <-ow.chDone:
			return
		case <-scanTimer.C:
			ow.checkIOs()
			ow.detect()
			for _, io := range ow.ios {
				err := io.read()
				if err != nil {
					log.Println("Error reading 1-wire io: ", err)
					busCount := ow.owNode.errorCount + 1
					ioCount := io.ioNode.errorCount + 1

					err = nats.SendNodePoint(ow.nc, ow.owNode.nodeID, data.Point{
						Type:  data.PointTypeErrorCount,
						Value: float64(busCount),
					}, false)

					err = nats.SendNodePoint(ow.nc, io.ioNode.nodeID, data.Point{
						Type:  data.PointTypeErrorCount,
						Value: float64(ioCount),
					}, false)
				}
			}
		}
	}
}
