package client

import (
	"log"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

type serialDev struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Port        string `point:"port"`
	Baud        string `point:"baud"`
	Debug       int    `point:"debug"`
}

type serialDevClient struct {
	nc            *nats.Conn
	config        serialDev
	stop          chan struct{}
	newPoints     chan []data.Point
	newEdgePoints chan []data.Point
}

func newSerialDevClient(nc *nats.Conn, config serialDev) Client {
	return &serialDevClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan []data.Point),
		newEdgePoints: make(chan []data.Point),
	}
}

// Start runs the main logic for this client and blocks until stopped
func (sd *serialDevClient) Start() error {
	log.Println("Starting serial client: ", sd.config.Description)
	for {
		select {
		case <-sd.stop:
			log.Println("Stopping serial client: ", sd.config.Description)
			return nil
		case pts := <-sd.newPoints:
			err := data.MergePoints(pts, &sd.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
			log.Printf("New config: %+v\n", sd.config)
		case pts := <-sd.newEdgePoints:
			err := data.MergeEdgePoints(pts, &sd.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
		}
	}
}

// Stop sends a signal to the Start function to exit
func (sd *serialDevClient) Stop(err error) {
	close(sd.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (sd *serialDevClient) Points(points []data.Point) {
	sd.newPoints <- points
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (sd *serialDevClient) EdgePoints(points []data.Point) {
	sd.newEdgePoints <- points
}
