package client

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"golang.org/x/exp/constraints"
)

// ShellyIOClient is a SIOT particle client
type ShellyIOClient struct {
	nc              *nats.Conn
	config          ShellyIo
	points          data.Points
	stop            chan struct{}
	newPoints       chan NewPoints
	newEdgePoints   chan NewPoints
	newShellyPoints chan NewPoints
	errorCount      int
	comps           []shellyComp
}

// NewShellyIOClient ...
func NewShellyIOClient(nc *nats.Conn, config ShellyIo) Client {
	// we need a copy of points with timestamps so we know when to send up new data
	ne, err := data.Encode(config)
	if err != nil {
		log.Println("Error encoding shelly config: ", err)
	}

	return &ShellyIOClient{
		nc:              nc,
		config:          config,
		comps:           shellyCompMap[config.Type],
		points:          ne.Points,
		stop:            make(chan struct{}),
		newPoints:       make(chan NewPoints),
		newEdgePoints:   make(chan NewPoints),
		newShellyPoints: make(chan NewPoints),
	}
}

// Run runs the main logic for this client and blocks until stopped
func (sioc *ShellyIOClient) Run() error {
	log.Println("Starting shelly IO client: ", sioc.config.Description)

	sampleRate := time.Second * 2
	sampleRateOffline := time.Minute * 10

	syncConfigTicker := time.NewTicker(sampleRateOffline)
	sampleTicker := time.NewTicker(sampleRate)

	if sioc.config.Offline {
		sampleTicker = time.NewTicker(sampleRateOffline)
	}

	if sioc.config.Disabled {
		sampleTicker.Stop()
	}

	shellyError := func() {
		sioc.errorCount++
		if !sioc.config.Offline && sioc.errorCount > 5 {
			log.Printf("Shelly device %v is offline", sioc.config.Description)
			sioc.config.Offline = true
			err := SendNodePoint(sioc.nc, sioc.config.ID, data.Point{
				Type: data.PointTypeOffline, Value: 1}, false)

			if err != nil {
				log.Println("ShellyIO: error sending node point: ", err)
			}
			sampleTicker = time.NewTicker(sampleRateOffline)
		}
	}

	shellyCommOK := func() {
		sioc.errorCount = 0
		if sioc.config.Offline {
			log.Printf("Shelly device %v is online", sioc.config.Description)
			sioc.config.Offline = false
			err := SendNodePoint(sioc.nc, sioc.config.ID, data.Point{
				Type: data.PointTypeOffline, Value: 0}, false)

			if err != nil {
				log.Println("ShellyIO: error sending node point: ", err)
			}
			sampleTicker = time.NewTicker(sampleRate)
		}
	}

	syncConfig := func() {
		config, err := sioc.config.getConfig()
		if err != nil {
			shellyError()
			log.Println("Error getting shelly IO settings: ", sioc.config.Desc(), err)
			return
		}

		shellyCommOK()

		if sioc.config.Description == "" && config.Name != "" {
			sioc.config.Description = config.Name
			err := SendNodePoint(sioc.nc, sioc.config.ID, data.Point{
				Type: data.PointTypeDescription, Text: config.Name}, false)
			if err != nil {
				log.Println("Error sending shelly io description: ", err)
			}
		} else if sioc.config.Description != config.Name {
			err := sioc.config.SetName(sioc.config.Description)
			if err != nil {
				log.Println("Error setting name on Shelly device: ", err)
			}
		}
	}

	syncConfig()

done:
	for {
		select {
		case <-sioc.stop:
			log.Println("Stopping shelly IO client: ", sioc.config.Description)
			break done
		case pts := <-sioc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &sioc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeDescription:
					syncConfig()
				case data.PointTypeDisabled:
					if p.Value == 0 {
						sampleTicker = time.NewTicker(sampleRate)
					} else {
						sampleTicker.Stop()
					}
				case data.PointTypeOffline:
					if p.Value == 0 {
						// defice is online
						// the discovery mechanism may have set the IO back online
						sampleTicker = time.NewTicker(sampleRate)
					} else {
						sampleTicker = time.NewTicker(sampleRateOffline)
					}
				}
			}

		case pts := <-sioc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &sioc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

		case <-syncConfigTicker.C:
			syncConfig()

		case <-sampleTicker.C:
			if sioc.config.Disabled {
				fmt.Println("Shelly IO is disabled, why am I ticking?")
				continue
			}
			points, err := sioc.config.GetStatus()
			if err != nil {
				log.Printf("Error getting status for %v: %v\n", sioc.config.Description, err)
				shellyError()
				break
			}

			if sioc.config.Control {
				switchCount := min(len(sioc.config.Switch), len(sioc.config.SwitchSet))
				for i := 0; i < switchCount; i++ {
					if sioc.config.Switch[i] != sioc.config.SwitchSet[i] {
						pts, err := sioc.config.SetOnOff("switch", i, sioc.config.SwitchSet[i])
						if err != nil {
							log.Printf("Error setting %v: %v\n", sioc.config.Description, err)
						}

						if len(pts) > 0 {
							points = append(points, pts...)
						} else {
							// get current status as the set did not return status
							points, err = sioc.config.GetStatus()
							if err != nil {
								log.Printf("Error getting status for %v: %v\n", sioc.config.Description, err)
								shellyError()
								break
							}
						}
					}
				}

				lightCount := min(len(sioc.config.Light), len(sioc.config.LightSet))
				for i := 0; i < lightCount; i++ {
					if sioc.config.Light[i] != sioc.config.LightSet[i] {
						pts, err := sioc.config.SetOnOff("light", i, sioc.config.LightSet[i])
						if err != nil {
							log.Printf("Error setting %v: %v\n", sioc.config.Description, err)
						}

						if len(pts) > 0 {
							points = append(points, pts...)
						} else {
							// get current status as the set did not return status
							points, err = sioc.config.GetStatus()
							if err != nil {
								log.Printf("Error getting status for %v: %v\n", sioc.config.Description, err)
								shellyError()
								break
							}
						}
					}
				}

			}

			shellyCommOK()

			newPoints := sioc.points.Merge(points, time.Minute*15)
			if len(newPoints) > 0 {
				err := data.MergePoints(sioc.config.ID, newPoints, &sioc.config)
				if err != nil {
					log.Println("shelly io: error merging newPoints: ", err)
				}
				err = SendNodePoints(sioc.nc, sioc.config.ID, newPoints, false)
				if err != nil {
					log.Println("shelly io: error sending newPoints: ", err)
				}
			}
		}
	}

	// clean up
	return nil
}

// Stop sends a signal to the Run function to exit
func (sioc *ShellyIOClient) Stop(_ error) {
	close(sioc.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (sioc *ShellyIOClient) Points(nodeID string, points []data.Point) {
	sioc.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (sioc *ShellyIOClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	sioc.newEdgePoints <- NewPoints{nodeID, parentID, points}
}

func min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}
