package client

import (
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/mdns"
	"github.com/simpleiot/simpleiot/data"
)

// Shelly describes the shelly client config
type Shelly struct {
	ID          string     `node:"id"`
	Parent      string     `node:"parent"`
	Description string     `point:"description"`
	Disabled    bool       `point:"disabled"`
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
	log.Println("Starting shelly client:", sc.config.Description)

	entriesCh := make(chan *mdns.ServiceEntry, 4)

	params := mdns.DefaultParams("_http._tcp")
	params.DisableIPv6 = true
	params.Entries = entriesCh

	scan := func() {
		err := mdns.Query(params)
		if err != nil {
			log.Println("mdns error:", err)
		}
	}

	go scan()

	scanTicker := time.NewTicker(time.Minute * 1)

done:
	for {
		select {
		case <-sc.stop:
			log.Println("Stopping shelly client:", sc.config.Description)
			break done
		case pts := <-sc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &sc.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeDisabled:
				}
			}

		case pts := <-sc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &sc.config)
			if err != nil {
				log.Println("error merging new points:", err)
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

				for i, io := range sc.config.IOs {
					if io.DeviceID == id {
						// already have this one
						// must set Origin because we are sending a point to another node
						// if we don't set origin, then the client manager will filter out
						// points to the client that owns the node
						found = true
						if io.IP != ip {
							err := SendNodePoint(sc.nc, io.ID, data.Point{
								Type:   data.PointTypeIP,
								Text:   ip,
								Origin: sc.config.ID,
							}, false)

							if err != nil {
								log.Println("Error setting io ip:", err)
							}
						}

						if io.Offline {
							err := SendNodePoint(sc.nc, io.ID, data.Point{
								Type:   data.PointTypeOffline,
								Value:  0,
								Origin: sc.config.ID,
							}, false)

							if err != nil {
								log.Println("Error setting io offline:", err)
							} else {
								sc.config.IOs[i].Offline = false
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
					IP:       ip,
				}

				ne, err := data.Encode(newIO)
				if err != nil {
					log.Println("Error encoding new shelly IO:", err)
					continue
				}

				addCompPoints := func(pType string, id int) {
					iString := strconv.Itoa(id)
					ne.Points = append(ne.Points, data.Point{Type: pType, Key: iString})
				}

				for _, comp := range shellyCompMap[typ] {
					switch comp.name {
					case "input":
						addCompPoints(data.PointTypeInput, comp.id)
					case "switch":
						addCompPoints(data.PointTypeSwitch, comp.id)
						addCompPoints(data.PointTypeSwitchSet, comp.id)
					case "light":
						addCompPoints(data.PointTypeLight, comp.id)
						addCompPoints(data.PointTypeLightSet, comp.id)
					}
				}

				err = SendNode(sc.nc, ne, sc.config.ID)
				if err != nil {
					log.Println("Error sending shelly IO:", err)
				}
			}
		}
	}

	// clean up
	scanTicker.Stop()
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
