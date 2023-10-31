package client

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// NodeTypeZMini indentifies a Zonit Mini node
var NodeTypeZMini = "zMini"

// ZMini represents a Zonit mini client
type ZMini struct {
	ID         string      `node:"id"`
	Parent     string      `node:"parent"`
	SerialPort []SerialDev `child:"serialDev"`
}

// ZMiniClient s a SIOT client used to manage Zonit mini devices
type ZMiniClient struct {
	nc            *nats.Conn
	config        ZMini
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
}

// NewZMiniClient ...
func NewZMiniClient(nc *nats.Conn, config ZMini) Client {
	return &ZMiniClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
	}
}

// Run the z-mini client
func (zm *ZMiniClient) Run() error {
	if len(zm.config.SerialPort) < 1 {
		// add a serial node
		sn := SerialDev{
			ID:          uuid.New().String(),
			Parent:      zm.config.ID,
			Description: "Serial port",
		}

		err := SendNodeType(zm.nc, sn, zm.config.ID)

		if err != nil {
			log.Println("Error creating serial child node: ", err)
			return fmt.Errorf("Error creating z-mini client, creating serial node: %v", err)
		}

		// client will restart here as we added a new node
	}

	return nil
}

// Stop sends a signal to the Run function to exit
func (zm *ZMiniClient) Stop(_ error) {
	close(zm.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (zm *ZMiniClient) Points(nodeID string, points []data.Point) {
	zm.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (zm *ZMiniClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	zm.newEdgePoints <- NewPoints{nodeID, parentID, points}
}
