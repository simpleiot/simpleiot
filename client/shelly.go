package client

import (
	"log"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/mdns"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Shelly describes the shelly client config
type Shelly struct {
	ID          string     `node:"id"`
	Parent      string     `node:"parent"`
	Description string     `point:"description"`
	Disable     bool       `point:"disable"`
	IOs         []ShellyIo `child:"shellyIo"`
}

// ShellyClient is a SIOT particle client
type ShellyClient struct {
	nc              *nats.Conn
	config          Shelly
	stop            chan struct{}
	newPoints       chan NewPoints
	newEdgePoints   chan NewPoints
	newShellyPoints chan NewPoints
}

// NewShellyClient ...
func NewShellyClient(nc *nats.Conn, config Shelly) Client {
	return &ShellyClient{
		nc:              nc,
		config:          config,
		stop:            make(chan struct{}),
		newPoints:       make(chan NewPoints),
		newEdgePoints:   make(chan NewPoints),
		newShellyPoints: make(chan NewPoints),
	}
}

// Run runs the main logic for this client and blocks until stopped
func (sc *ShellyClient) Run() error {
	log.Println("Starting shelly client: ", sc.config.Description)

	entriesCh := make(chan *mdns.ServiceEntry, 4)

	scan := func() {
		err := mdns.Lookup("_http._tcp", entriesCh)
		if err != nil {
			log.Println("mdns error: ", err)
		}
	}

	go scan()

	scanTicker := time.NewTicker(time.Minute * 1)

done:
	for {
		select {
		case <-sc.stop:
			log.Println("Stopping shelly client: ", sc.config.Description)
			break done
		case pts := <-sc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &sc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeDisable:
				}
			}

		case pts := <-sc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &sc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

		case <-scanTicker.C:
			go scan()

		case e := <-entriesCh:
			typ, id := shellyScanHost(e.Host)
			if len(typ) > 0 {
				found := false

				var ip string
				if e.AddrV4 != nil {
					ip = e.AddrV4.String()
				} else if e.AddrV6 != nil {
					ip = e.AddrV6.String()
				}

				for _, io := range sc.config.IOs {
					if io.DeviceID == id {
						// already have this one
						found = true
						if io.IP != ip {
							err := SendNodePoint(sc.nc, io.ID, data.Point{
								Type: data.PointTypeIP,
								Text: ip,
							}, false)

							if err != nil {
								log.Println("Error setting io ip: ", err)
							}
						}
						break
					}
				}
				if found {
					break
				}

				newIO := ShellyIo{
					ID:       uuid.New().String(),
					DeviceID: id,
					Parent:   sc.config.ID,
					Type:     typ,
				}

				err := SendNodeType(sc.nc, newIO, sc.config.ID)
				if err != nil {
					log.Println("Error sending shelly IO: ", err)
				}
			}
		}
	}

	// clean up
	return nil
}

// Stop sends a signal to the Run function to exit
func (sc *ShellyClient) Stop(_ error) {
	close(sc.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (sc *ShellyClient) Points(nodeID string, points []data.Point) {
	sc.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (sc *ShellyClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	sc.newEdgePoints <- NewPoints{nodeID, parentID, points}
}

var reShellyHost = regexp.MustCompile("(?i)shelly(.*)-(.*).local")

func shellyScanHost(host string) (string, string) {
	m := reShellyHost.FindStringSubmatch(host)
	if len(m) < 3 {
		return "", ""
	}

	return m[1], m[2]
}
