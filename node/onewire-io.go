package node

import (
	"log"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

type oneWireIO struct {
	ioNode   *oneWireIONode
	sub      *natsgo.Subscription
	lastSent time.Time
}

func newOneWireIO(nc *natsgo.Conn, node *oneWireIONode, chPoint chan<- pointWID) (*oneWireIO, error) {
	io := &oneWireIO{
		ioNode: node,
	}

	var err error
	io.sub, err = nc.Subscribe("node."+io.ioNode.nodeID+".points", func(msg *natsgo.Msg) {
		points, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			// FIXME, send over channel
			log.Println("Error decoding node data: ", err)
			return
		}

		for _, p := range points {
			chPoint <- pointWID{io.ioNode.nodeID, p}
		}
	})

	if err != nil {
		return nil, err
	}

	return io, nil
}

// Stop io
func (io *oneWireIO) Stop() {
	if io.sub != nil {
		err := io.sub.Unsubscribe()
		if err != nil {
			log.Println("Error unsubscribing from IO: ", err)
		}
	}
}
