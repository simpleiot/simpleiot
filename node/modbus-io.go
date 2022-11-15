package node

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// ModbusIO represents the state of a managed modbus io
type ModbusIO struct {
	ioNode   *ModbusIONode
	sub      *nats.Subscription
	lastSent time.Time
}

// NewModbusIO creates a new modbus IO
func NewModbusIO(nc *nats.Conn, node *ModbusIONode, chPoint chan<- pointWID) (*ModbusIO, error) {
	io := &ModbusIO{
		ioNode: node,
	}

	var err error
	io.sub, err = nc.Subscribe("p."+io.ioNode.nodeID, func(msg *nats.Msg) {
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
func (io *ModbusIO) Stop() {
	if io.sub != nil {
		err := io.sub.Unsubscribe()
		if err != nil {
			log.Println("Error unsubscribing from IO: ", err)
		}
	}
}
