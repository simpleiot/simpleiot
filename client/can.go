package client

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/simpleiot/canparse"
	"github.com/simpleiot/simpleiot/data"
	"go.einride.tech/can"
	"go.einride.tech/can/pkg/socketcan"
)

// CanBus represents a CAN socket config. The name matches the front-end node type "canBus" to link the two so
// that when a canBus node is created on the frontend the client manager knows to start a CanBus client.
type CanBus struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Interface   string `point:"interface"`
	DbFilePath  string `point:"dbFilePath"`
}

// CanBusClient is a SIOT client used to communicate on a CAN bus
type CanBusClient struct {
	nc            *nats.Conn
	config        CanBus
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
	wrSeq         byte
	lastSendStats time.Time
	natsSub       string
}

// NewCanBusClient ...
func NewCanBusClient(nc *nats.Conn, config CanBus) Client {
	return &CanBusClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		wrSeq:         0,
		lastSendStats: time.Time{},
		natsSub:       SubjectNodePoints(config.ID),
	}
}

// Start runs the main logic for this client and blocks until stopped
// There are several main aspects of the CAN bus client
//
//   - the listener function is a process that recieves CAN bus frames from the Linux
//	   SocketCAN socket and sends the frames out on the canMsgRx channel
//
//   - when a frame is recieved on the canMsgRx channel in the main loop, it is decoded
//	   and a point is sent out for each canparse.Signal in the frame. The key of each point
//     contains the message name, signal name, and signal units
//
func (cb *CanBusClient) Start() error {
	log.Println("CanBusClient: Starting CAN bus client: ", cb.config.Description)

	// Setup CAN Database
	db := &canparse.Database{}
	err := db.ReadKcd(cb.config.DbFilePath)
	if err != nil {
		log.Println(errors.Wrap(err, "CanBusClient: Error parsing KCD file:"))
	} else {
		for _, b := range db.Busses {
			for _, m := range b.Messages {
				for _, s := range m.Signals {
					log.Printf("CanBusClient: read msg %X sig %v: start=%v len=%v scale=%v offset=%v unit=%v",
						m.Id, s.Name, s.Start, s.Length, s.Scale, s.Offset, s.Unit)
				}
			}
		}
	}

	canMsgRx := make(chan can.Frame)

	conn, err := socketcan.DialContext(context.Background(), "can", cb.config.Interface)
	if err != nil {
		log.Println(errors.Wrap(err, "CanBusClient: error dialing socketcan context"))
	}
	recv := socketcan.NewReceiver(conn)

	listener := func() {
		for recv.Receive() {
			frame := recv.Frame()
			canMsgRx <- frame
		}
	}

	go listener()

	for {
		select {
		case <-cb.stop:
			log.Println("CanBusClient: stopping CAN bus client: ", cb.config.Description)
			return nil

		case frame := <-canMsgRx:

			log.Println("CanBusClient: got", frame.String())

			msg, err := canparse.DecodeMessage(frame, db)
			if err != nil {
				log.Println(errors.Wrap(err, "CanBusClient: error decoding CAN message"))
			}

			points := make(data.Points, len(msg.Signals))
			for i, sig := range msg.Signals {
				points[i].Key = fmt.Sprintf("%v.%v(%v)", msg.Name, sig.Name, sig.Unit)
				points[i].Time = time.Now()
				points[i].Value = float64(sig.Value)
				log.Println("CanBusClient: created point", points[i].Key, points[i].Value)
			}

			// Send the points
			if len(points) > 0 {
				err = SendPoints(cb.nc, cb.natsSub, points, false)
				if err != nil {
					log.Println(errors.Wrap(err, "CanBusClient: error sending points received from CAN bus: "))
				} else {
					log.Println("CanBusClient: successfully sent points")
				}
			}

		case pts := <-cb.newPoints:
			for _, p := range pts.Points {
				if p.Type == data.PointTypePort ||
					p.Type == data.PointTypeBaud ||
					p.Type == data.PointTypeDisable {
					break
				}

				if p.Type == data.PointTypeDisable {
					if p.Value == 0 {
					}
				}
			}

			err := data.MergePoints(pts.ID, pts.Points, &cb.config)
			if err != nil {
				log.Println("CanBusClient: error merging new points: ", err)
			}

		case pts := <-cb.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &cb.config)
			if err != nil {
				log.Println("CanBusClient: error merging new points: ", err)
			}

			// TODO need to send edge points to CAN bus, not implemented yet
		}
	}
}

// Stop sends a signal to the Start function to exit
func (cb *CanBusClient) Stop(err error) {
	close(cb.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (cb *CanBusClient) Points(nodeID string, points []data.Point) {
	cb.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (cb *CanBusClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	cb.newEdgePoints <- NewPoints{nodeID, parentID, points}
}
