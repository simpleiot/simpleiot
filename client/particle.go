package client

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/donovanhide/eventsource"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Get Particle.io data using their event API. See:
// https://docs.particle.io/reference/cloud-apis/api/#get-a-stream-of-events

const particleEventURL string = "https://api.particle.io/v1/devices/events/"

// ParticleEvent from particle
type ParticleEvent struct {
	Data      string    `json:"data"`
	TTL       uint32    `json:"ttl"`
	Timestamp time.Time `json:"published_at"`
	CoreID    string    `json:"coreid"`
}

// Particle represents the configuration for the SIOT Particle client
type Particle struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Disable     bool   `point:"disable"`
	AuthToken   string `point:"authToken"`
}

// ParticleClient is a SIOT particle client
type ParticleClient struct {
	nc                *nats.Conn
	config            Particle
	stop              chan struct{}
	newPoints         chan NewPoints
	newEdgePoints     chan NewPoints
	newParticlePoints chan NewPoints
}

type particlePoint struct {
	ID    string  `json:"id"`
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

func (pp *particlePoint) toPoint() data.Point {
	return data.Point{
		Key:   pp.ID,
		Type:  pp.Type,
		Value: pp.Value,
	}
}

// NewParticleClient ...
func NewParticleClient(nc *nats.Conn, config Particle) Client {
	return &ParticleClient{
		nc:                nc,
		config:            config,
		stop:              make(chan struct{}),
		newPoints:         make(chan NewPoints),
		newEdgePoints:     make(chan NewPoints),
		newParticlePoints: make(chan NewPoints),
	}
}

// Run runs the main logic for this client and blocks until stopped
func (pc *ParticleClient) Run() error {
	log.Println("Starting particle client: ", pc.config.Description)

	closeReader := make(chan struct{})  // is closed to close reader
	readerClosed := make(chan struct{}) // struct{} is sent when reader exits
	var readerRunning bool              // indicates reader is running

	particleReader := func() {
		log.Println("CLIFF: particle reader started")
		defer func() {
			log.Println("CLIFF: particle reader exitted")
			readerClosed <- struct{}{}
		}()

		urlAuth := particleEventURL + "sample" + "?access_token=" + pc.config.AuthToken

		stream, err := eventsource.Subscribe(urlAuth, "")

		if err != nil {
			log.Println("Particle subscription error: ", err)
			return
		}

		for {
			select {
			case event := <-stream.Events:
				var pEvent ParticleEvent
				err := json.Unmarshal([]byte(event.Data()), &pEvent)
				if err != nil {
					log.Println("Got error decoding particle event: ", err)
					continue
				}

				var pPoints []particlePoint
				err = json.Unmarshal([]byte(pEvent.Data), &pPoints)
				if err != nil {
					log.Println("error decoding Particle samples: ", err)
					continue
				}

				points := make(data.Points, len(pPoints))

				for i, p := range pPoints {
					points[i] = p.toPoint()
					points[i].Time = pEvent.Timestamp
				}

				fmt.Println("CLIFF: particle points: ", points)

				err = SendNodePoints(pc.nc, pc.config.ID, points, false)
				if err != nil {
					log.Println("Particle error sending points: ", err)
				}

			case err := <-stream.Errors:
				log.Println("Particle error: ", err)

			case <-closeReader:
				log.Println("Exiting particle reader")
				return
			}
		}
	}

	checkTime := time.Minute
	checkReader := time.NewTicker(checkTime)

	startReader := func() {
		if readerRunning {
			return
		}
		readerRunning = true
		go particleReader()
		checkReader.Stop()
	}

	stopReader := func() {
		if readerRunning {
			closeReader <- struct{}{}
			readerRunning = false
		}
	}

	startReader()

done:
	for {
		select {
		case <-pc.stop:
			log.Println("Stopping db client: ", pc.config.Description)
			break done
		case pts := <-pc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &pc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeAuthToken:
					stopReader()
					startReader()
				case data.PointTypeDisable:
					if p.Value == 1 {
						stopReader()
					} else {
						startReader()
					}
				}
			}

		case pts := <-pc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &pc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

		case <-readerClosed:
			readerRunning = false
			checkReader.Reset(checkTime)

		case <-checkReader.C:
			startReader()
		}
	}

	// clean up
	stopReader()
	return nil
}

// Stop sends a signal to the Run function to exit
func (pc *ParticleClient) Stop(err error) {
	close(pc.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (pc *ParticleClient) Points(nodeID string, points []data.Point) {
	pc.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (pc *ParticleClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	pc.newEdgePoints <- NewPoints{nodeID, parentID, points}
}
