package node

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/nats"
)

type oneWireIO struct {
	nc       *natsgo.Conn
	ioNode   *oneWireIONode
	path     string
	sub      *natsgo.Subscription
	lastSent time.Time
}

func newOneWireIO(nc *natsgo.Conn, node *oneWireIONode, chPoint chan<- pointWID) (*oneWireIO, error) {
	io := &oneWireIO{
		nc:     nc,
		path:   fmt.Sprintf("/sys/bus/w1/devices/%v/temperature", node.id),
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
func (io *oneWireIO) stop() {
	if io.sub != nil {
		err := io.sub.Unsubscribe()
		if err != nil {
			log.Println("Error unsubscribing from IO: ", err)
		}
	}
}

func (io *oneWireIO) point(p data.Point) error {
	// handle IO changes
	switch p.Type {
	case data.PointTypeID:
		io.ioNode.id = p.Text
	case data.PointTypeDescription:
		io.ioNode.description = p.Text
	case data.PointTypeValue:
		io.ioNode.value = p.Value
	case data.PointTypeDisable:
		io.ioNode.disable = data.FloatToBool(p.Value)
	case data.PointTypeErrorCount:
		io.ioNode.errorCount = int(p.Value)
	case data.PointTypeErrorCountReset:
		io.ioNode.errorCountReset = data.FloatToBool(p.Value)
		if io.ioNode.errorCountReset {
			p := data.Points{
				{Type: data.PointTypeErrorCount, Value: 0},
				{Type: data.PointTypeErrorCountReset, Value: 0},
			}

			err := nats.SendNodePoints(io.nc, io.ioNode.nodeID, p, true)
			if err != nil {
				log.Println("Send point error: ", err)
			}
		}

	default:
		log.Println("1-wire: unhandled io point: ", p)
	}

	return nil
}

func (io *oneWireIO) read() error {
	if io.ioNode.disable {
		return nil
	}

	d, err := ioutil.ReadFile(io.path)
	if err != nil {
		return err
	}

	vRaw, err := strconv.Atoi(strings.TrimSpace(string(d)))
	if err != nil {
		return err
	}

	v := float64(vRaw) / 1000

	if v != io.ioNode.value || time.Since(io.lastSent) > time.Minute*10 {
		io.ioNode.value = v
		err = nats.SendNodePoint(io.nc, io.ioNode.nodeID, data.Point{
			Type:  data.PointTypeValue,
			Value: v,
		}, false)
		io.lastSent = time.Now()
	}

	return err
}
