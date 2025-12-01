package client

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
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
	BitRate             string `point:"bitRate"`
	MsgsInDb            int    `point:"msgsInDb"`
	SignalsInDb         int    `point:"signalsInDb"`
	MsgsRecvdDb         int    `point:"msgsRecvdDb"`
	MsgsRecvdDbReset    bool   `point:"msgsRecvdDbReset"`
	MsgsRecvdOther      int    `point:"msgsRecvdOther"`
	MsgsRecvdOtherReset bool   `point:"msgsRecvdOtherReset"`
	Databases           []File `child:"file"`
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

// Run the main logic for this client and blocks until stopped
// There are several main aspects of the CAN bus client
//
//   - the listener function is a process that receives CAN bus frames from the
//     Linux SocketCAN socket and sends the frames out on the canMsgRx channel
//
//   - when a frame is received on the canMsgRx channel in the main loop, it is
//     decoded and a point is sent out for each canparse.Signal in the frame.
//     The key of each point contains the message name, signal name, and signal
//     units
func (cb *CanBusClient) Run() error {
	log.Println("CanBusClient: Starting CAN bus client:", cb.config.Description)

	db := &canparse.Database{}

	sendDbStats := func(msgs, signals int) {
		points := data.Points{
			data.Point{
				Time:  time.Now(),
				Type:  data.PointTypeMsgsInDb,
				Value: float64(msgs),
			},
			data.Point{
				Time:  time.Now(),
				Type:  data.PointTypeSignalsInDb,
				Value: float64(signals),
			},
		}

		err := SendPoints(cb.nc, cb.natsSub, points, false)
		if err != nil {
			log.Println(errors.Wrap(err, "CanBusClient: error CAN db stats: "))
		}
	}

	readDb := func() {
		cb.config.MsgsInDb = 0
		cb.config.SignalsInDb = 0
		db.Clean()
		for _, dbFile := range cb.config.Databases {
			err := db.ReadBytes([]byte(dbFile.Data), dbFile.Name)
			if err != nil {
				log.Println(errors.Wrap(err, "CanBusClient: Error parsing database file"))
				sendDbStats(0, 0)
				return
			}
			for _, b := range db.Busses {
				cb.config.MsgsInDb += len(b.Messages)
				for _, m := range b.Messages {
					cb.config.SignalsInDb += len(m.Signals)
					/*
						for _, s := range m.Signals {
							log.Printf("CanBusClient: read msg %X sig %v: start=%v len=%v scale=%v offset=%v unit=%v",
								m.Id, s.Name, s.Start, s.Length, s.Scale, s.Offset, s.Unit)
						}
					*/
				}
			}
		}
		sendDbStats(cb.config.MsgsInDb, cb.config.SignalsInDb)
	}

	readDb()

	canMsgRx := make(chan can.Frame)

	var ctx context.Context
	var cancelContext context.CancelFunc

	// setupDev bringDownDev must be called before every call of setupDev //
	// except for the first call
	setupDev := func() {

		// Set up the socketCan interface
		iface, err := net.InterfaceByName(cb.config.Device)
		if err != nil {
			log.Println(errors.Wrap(err,
				"CanBusClient: socketCan interface not found"))

			return
		}
		if iface.Flags&net.FlagUp == 0 {
			err = exec.Command(
				"ip", "link", "set", cb.config.Device, "up", "type",
				"can", "bitrate", cb.config.BitRate).Run()
			if err != nil {
				log.Println(
					errors.Wrap(err, fmt.Sprintf("CanBusClient: error bringing up socketCan interface with: device=%v, bitrate=%v",
						cb.config.Device, cb.config.BitRate)))

			} else {
				log.Println(
					"CanBusClient: bringing up socketCan interface with:",
					cb.config.Device, cb.config.BitRate)
			}
		}

		// Connect to the socketCan interface
		ctx, cancelContext = context.WithCancel(context.Background())
		_ = cancelContext
		conn, err := socketcan.DialContext(ctx, "can", cb.config.Device)
		if err != nil {
			log.Println(errors.Wrap(err, "CanBusClient: error dialing socketcan context"))
			return
		}
		recv := socketcan.NewReceiver(conn)

		// Listen on the socketCan interface
		listener := func() {
			for recv.Receive() {
				frame := recv.Frame()
				canMsgRx <- frame
			}
		}
		go listener()
	}

	setupDev()

	bringDownDev := func() {
		if cancelContext != nil {
			cancelContext()
		}
	}

	for {
		select {
		case <-cb.stop:
			log.Println("CanBusClient: stopping CAN bus client:", cb.config.Description)
			bringDownDev()
			return nil

		case frame := <-canMsgRx:

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
				points[i].Type = data.PointTypeValue
				points[i].Key = fmt.Sprintf("%v.%v[%v]",
					msg.Name, sig.Name, sig.Unit)
				points[i].Time = time.Now()
				points[i].Value = float64(sig.Value)
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
				log.Println("CanBusClient: error merging new points:", err)
			}

			// Update CAN devices and databases with new information
			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeDevice:
					bringDownDev()
					setupDev()
				case data.PointTypeData:
					readDb()
				case data.PointTypeDisabled:
					if p.Value == 0 {
						bringDownDev()
					}
				}
			}

			// Reset db msgs received counter
			if cb.config.MsgsRecvdDbReset {
				points := data.Points{
					{Time: time.Now(), Type: data.PointTypeMsgsRecvdDb, Value: 0},
					{Time: time.Now(), Type: data.PointTypeMsgsRecvdDbReset, Value: 0},
				}
				err = SendPoints(cb.nc, cb.natsSub, points, false)
				if err != nil {
					log.Println("Error resetting CAN message received count:", err)
				}

				cb.config.MsgsRecvdDbReset = false
				cb.config.MsgsRecvdDb = 0
			}

			// Reset other msgs received counter
			if cb.config.MsgsRecvdOtherReset {
				points := data.Points{
					{Time: time.Now(), Type: data.PointTypeMsgsRecvdOther, Value: 0},
					{Time: time.Now(), Type: data.PointTypeMsgsRecvdOtherReset, Value: 0},
				}
				err = SendPoints(cb.nc, cb.natsSub, points, false)
				if err != nil {
					log.Println("Error resetting CAN message received count:", err)
				}

				cb.config.MsgsRecvdOtherReset = false
				cb.config.MsgsRecvdOther = 0
			}

		case pts := <-cb.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &cb.config)
			if err != nil {
				log.Println("CanBusClient: error merging new points:", err)
			}

			// TODO need to send edge points to CAN bus, not implemented yet
		}
	}
}

// Stop sends a signal to the Run function to exit
func (cb *CanBusClient) Stop(_ error) {
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
