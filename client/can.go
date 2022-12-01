package client

import (
	"log"
	"time"
	"net"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/simpleiot/simpleiot/data"
	//"golang.org/x/sys/unix"

	"github.com/go-daq/canbus"
)

// CanBus represents a CAN socket config. The name matches the front-end node type "canBus" to link the two so
// that when a canBus node is created on the frontend the client manager knows to start a CanBus client.
type CanBus struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Interface   string `point:"interface"`
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
//   - the listener function is a process that recieves CAN bus frames from the Linux SocketCAN socket
//     and sends the frames out on the canMsgRx channel
//
//   - the openPort function brings up a Linux network interface (if the interface of that name is not already up),
//     sets up CAN filters on the SocketCAN socket, and binds the socket to the interface. It also starts the listener
//     function in a go routine.
//
//   - when a frame is recieved on the canMsgRx channel in the main loop, it is decoded and a point is sent out for each
//     J1939 SPN in the frame. The key of each point contains the PGN, SPN, and description of the SPN
func (cb *CanBusClient) Start() error {
	log.Println("CanBusClient: Starting CAN bus client: ", cb.config.Description)

	socket, err := canbus.New()
	if err != nil {
		log.Println(errors.Wrap(err, "CanBusClient: Error creating Socket object"))
	}

	canMsgRx := make(chan canbus.Frame)

	closePort := func() {
		socket.Close()
	}

	listener := func() {
		for {
			frame, err := socket.Recv()
			if err != nil {
				log.Println(errors.Wrap(err, "CanBusClient: Error recieving CAN frame"))
			}
			canMsgRx <- frame
		}
	}

	openPort := func() {

		iface, err := net.InterfaceByName(cb.config.Interface)
		_ = iface
		if err != nil {
			log.Println(errors.Wrap(err, "CanBusClient: CAN interface not found"))
		}

		// TODO: figure out how to handle interface
		/*
		//if iface.Flags&net.FlagUp == 0 {
		// bring up CAN interface
		err = exec.Command("ip", "link", "set", cb.config.Interface, "type",
			"can", "bitrate", cb.config.BusSpeed).Run()
		log.Println("Bringing up IP link with:", cb.config.Interface, cb.config.BusSpeed)
		if err != nil {
			log.Println(errors.Wrap(err, "Error configuring internal CAN interface"))
		}

		err = exec.Command("ip", "link", "set", cb.config.Interface, "up").Run()
		if err != nil {
			log.Println(errors.Wrap(err, "Error bringing up internal can interface"))
		}
		*/
		/*
			} else {
				// Handle case where interface is already up and bus speed may be wrong
				log.Println("Error bringing up internal CAN interface, already set up.")
			}
		*/
		// TODO: figure out a way to set accepted CAN id's for filtering and decoding
		/*
		var filters []unix.CanFilter
		for _, id := range cb.config.CanIdsAccepted {
			filters = append(filters, unix.CanFilter{Id: id, Mask: unix.CAN_EFF_MASK})
			log.Println("CanBusClient: set filter {Id: %X, Mask: %X}", id, unix.CAN_EFF_MASK)
		}
		socket.SetFilters(filters[:])
		*/

		err = socket.Bind(cb.config.Interface)
		if err != nil {
			log.Println(errors.Wrap(err, "Error binding to CAN interface"))
		}
		go listener()
	}

	openPort()

	for {
		select {
		case <-cb.stop:
			log.Println("CanBusClient: stopping CAN bus client: ", cb.config.Description)
			closePort()
			return nil

		case frame := <-canMsgRx:

			log.Printf("CanBusClient: got %X, data length: %v\n", frame.ID, len(frame.Data))

			points := make(data.Points, 2)

			// FIXME decode data based on information in config
			points[0].Time = time.Now()
			points[1].Time = time.Now()
			points[0].Key = "FE48-1862-WheelBasedSpeed"
			points[1].Key = "FE48-1864-WheelBasedDirection"
			points[0].Value = 0
			points[1].Value = 0

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
						closePort()
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
