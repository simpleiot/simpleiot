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
	ID                  string `node:"id"`
	Parent              string `node:"parent"`
	Description         string `point:"description"`
	Device              string `point:"device"`
	MsgsInDb            int    `point:"msgsInDb"`
	SignalsInDb         int    `point:"signalsInDb"`
	MsgsRecvdDb         int    `point:"msgsRecvdDb"`
	MsgsRecvdDbReset    bool   `point:"msgsRecvdDbReset"`
	MsgsRecvdOther      int    `point:"msgsRecvdOther"`
	MsgsRecvdOtherReset bool   `point:"msgsRecvdOtherReset"`
	Databases           []File `child:"file"`
}

// File represents a CAN database file in common formats such as KCD and DBC.
type File struct {
	Name string `point:"name"`
	Data string `point:"data"`
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

// NewCanBusClient returns a new CanBusClient with a NATS connection and a config
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

	var db *canparse.Database = &canparse.Database{}

	readDb := func() {
		cb.config.MsgsInDb = 0
		cb.config.SignalsInDb = 0
		db.Clean()
		for _, dbFile := range cb.config.Databases {
			err := db.ReadBytes([]byte(dbFile.Data), dbFile.Name)
			if err != nil {
				log.Println(errors.Wrap(err, "CanBusClient: Error parsing database file"))
				return
			} else {
				for _, b := range db.Busses {
					cb.config.MsgsInDb += len(b.Messages)
					for _, m := range b.Messages {
						cb.config.SignalsInDb += len(m.Signals)
						for _, s := range m.Signals {
							log.Printf("CanBusClient: read msg %X sig %v: start=%v len=%v scale=%v offset=%v unit=%v",
								m.Id, s.Name, s.Start, s.Length, s.Scale, s.Offset, s.Unit)
						}
					}
				}
			}
		}
		points := data.Points{
			data.Point{
				Time:  time.Now(),
				Type:  data.PointTypeMsgsInDb,
				Value: float64(cb.config.MsgsInDb),
			},
			data.Point{
				Time:  time.Now(),
				Type:  data.PointTypeSignalsInDb,
				Value: float64(cb.config.SignalsInDb),
			},
		}
		err := SendPoints(cb.nc, cb.natsSub, points, false)
		if err != nil {
			log.Println(errors.Wrap(err, "CanBusClient: error sending points received from CAN bus: "))
		}
	}

	readDb()

	canMsgRx := make(chan can.Frame)

	bringDownDev := func() {}

	setupDev := func() {
		bringDownDev()
		conn, err := socketcan.DialContext(context.Background(), "can", cb.config.Device)
		if err != nil {
			log.Println(errors.Wrap(err, "CanBusClient: error dialing socketcan context"))
			return
		}
		recv := socketcan.NewReceiver(conn)

		listener := func() {
			for recv.Receive() {
				frame := recv.Frame()
				canMsgRx <- frame
			}
		}
		go listener()
	}

	setupDev()

	for {
		select {
		case <-cb.stop:
			log.Println("CanBusClient: stopping CAN bus client: ", cb.config.Description)
			return nil

		case frame := <-canMsgRx:

			log.Println("CanBusClient: got", frame.String())

			// Decode the can message based on database
			msg, err := canparse.DecodeMessage(frame, db)
			if err != nil {
				cb.config.MsgsRecvdOther++
			} else {
				cb.config.MsgsRecvdDb++
			}

			// Populate points representing the decoded CAN data
			points := make(data.Points, len(msg.Signals))
			for i, sig := range msg.Signals {
				points[i].Key = fmt.Sprintf("%v.%v[%v]", msg.Name, sig.Name, sig.Unit)
				points[i].Time = time.Now()
				points[i].Value = float64(sig.Value)
				log.Println("CanBusClient: created point", points[i].Key, points[i].Value)
			}

			// Populate points to update CAN client stats
			points = append(points,
				data.Point{
					Time:  time.Now(),
					Type:  data.PointTypeMsgsRecvdDb,
					Value: float64(cb.config.MsgsRecvdDb),
				})
			points = append(points,
				data.Point{
					Time:  time.Now(),
					Type:  data.PointTypeMsgsRecvdOther,
					Value: float64(cb.config.MsgsRecvdOther),
				})

			// Send the points
			if len(points) > 0 {
				err = SendPoints(cb.nc, cb.natsSub, points, false)
				if err != nil {
					log.Println(errors.Wrap(err, "CanBusClient: error sending points received from CAN bus: "))
				}
			}

		case pts := <-cb.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &cb.config)
			if err != nil {
				log.Println("CanBusClient: error merging new points: ", err)
			}

			// Update CAN devices and databases with new information
			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeDevice:
					setupDev()
				case data.PointTypeData:
					readDb()
				case data.PointTypeName:
					log.Println("CanBusClient, point name text:", p.Text, "value:", p.Value)
					log.Println("CanBusClient, config name:", cb.config.Databases[0].Name)
					// FIXME shouldn't have to do this manually
					if len(cb.config.Databases) > 0 {
						cb.config.Databases[0].Name = p.Text
					}
					log.Println("CanBusClient, config name:", cb.config.Databases[0].Name)
					readDb()
				case data.PointTypeDisable:
					if p.Value == 0 {
						bringDownDev()
					}
				}
			}

			// Reset db msgs recieved counter
			if cb.config.MsgsRecvdDbReset {
				points := data.Points{
					{Time: time.Now(), Type: data.PointTypeMsgsRecvdDb, Value: 0},
					{Time: time.Now(), Type: data.PointTypeMsgsRecvdDbReset, Value: 0},
				}
				err = SendPoints(cb.nc, cb.natsSub, points, false)
				if err != nil {
					log.Println("Error resetting CAN message recieved count: ", err)
				}

				cb.config.MsgsRecvdDbReset = false
				cb.config.MsgsRecvdDb = 0
			}

			// Reset other msgs recieved counter
			if cb.config.MsgsRecvdOtherReset {
				points := data.Points{
					{Time: time.Now(), Type: data.PointTypeMsgsRecvdOther, Value: 0},
					{Time: time.Now(), Type: data.PointTypeMsgsRecvdOtherReset, Value: 0},
				}
				err = SendPoints(cb.nc, cb.natsSub, points, false)
				if err != nil {
					log.Println("Error resetting CAN message recieved count: ", err)
				}

				cb.config.MsgsRecvdOtherReset = false
				cb.config.MsgsRecvdOther = 0
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
