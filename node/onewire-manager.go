package node

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// oneWireManager is responsible for finding new busses and sync database state
// with what is here
type oneWireManager struct {
	nc         *nats.Conn
	busses     map[string]*oneWire
	rootNodeID string
}

func newOneWireManager(nc *nats.Conn, rootNodeID string) *oneWireManager {
	return &oneWireManager{
		nc:         nc,
		busses:     make(map[string]*oneWire),
		rootNodeID: rootNodeID,
	}
}

var reBusMaster = regexp.MustCompile(`w1_bus_master(\d+)`)

func (owm *oneWireManager) update() error {
	nodes, err := client.GetNodes(owm.nc, owm.rootNodeID, "all", data.NodeTypeOneWire, false)
	if err != nil {
		return err
	}

	found := make(map[string]bool)

	for _, node := range nodes {
		found[node.ID] = true
		_, ok := owm.busses[node.ID]
		if !ok {
			var err error
			bus, err := newOneWire(owm.nc, node)
			if err != nil {
				log.Println("Error creating new modbus:", err)
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
			log.Println("removing onewire bus:", bus.owNode.description)
			bus.stop()
			delete(owm.busses, id)
		}
	}

	// detect one wire busses
	dirs, _ := filepath.Glob("/sys/bus/w1/devices/w1_bus_master*")

	for _, dir := range dirs {
		f, _ := os.Stat(dir)
		if f.IsDir() {
			ms := reBusMaster.FindStringSubmatch(dir)
			if len(ms) < 2 {
				continue
			}

			index, err := strconv.Atoi(ms[1])

			if err != nil {
				log.Println("Error extracting 1-wire bus number:", err)
			}

			// loop through busses and make sure it exists
			found := false
			for _, b := range owm.busses {
				if b.owNode.index == index {
					found = true
					break
				}
			}

			if !found {
				log.Printf("Adding 1-wire bus #%v\n", index)

				n := data.NodeEdge{
					Type:   data.NodeTypeOneWire,
					Parent: owm.rootNodeID,
					Points: data.Points{
						data.Point{
							Type:  data.PointTypeIndex,
							Value: float64(index),
						},
						data.Point{
							Type: data.PointTypeDescription,
							Text: "New bus, please edit",
						},
					},
				}

				err := client.SendNode(owm.nc, n, "")
				if err != nil {
					log.Println("Error sending new 1-wire node:", err)
				}
			}
		}
	}

	return nil

}
