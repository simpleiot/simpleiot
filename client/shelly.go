package client

import (
	"log"
	"regexp"
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

	scan := func() {
		params := mdns.DefaultParams("_http._tcp")
		params.DisableIPv6 = true
		params.Entries = entriesCh
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

		case ePtr := <-entriesCh:
			e := *ePtr // copy to avoid data race with mdns goroutine
			if !shellyHost(e.Host) {
				break
			}

			var ip string
			if e.AddrV4 != nil {
				ip = e.AddrV4.String()
			} else if e.AddrV6 != nil {
				ip = e.AddrV6.String()
			}
			if ip == "" {
				break
			}

			// Ask the device what it is rather than reading its model out of
			// the mDNS hostname. This is also what tells us the generation,
			// and it works for a device this code has never heard of.
			di, err := shellyGetDeviceInfo(ip)
			if err != nil {
				log.Println("Error getting shelly device info:", ip, err)
				break
			}

			id := shellyDeviceID(e.Host, di)
			if id == "" {
				break
			}

			found := false
			for i, io := range sc.config.IOs {
				if io.DeviceID != id {
					continue
				}
				// already have this one
				// must set Origin because we are sending a point to another node
				// if we don't set origin, then the client manager will filter out
				// points to the client that owns the node
				found = true
				if io.IP != ip {
					err := SendNodePoint(sc.nc, io.ID, func() data.Point {
						p := data.NewPointString(data.PointTypeIP, "", ip)
						p.Origin = sc.config.ID
						return p
					}(), false)

					if err != nil {
						log.Println("Error setting io ip:", err)
					}
				}

				if io.Offline {
					err := SendNodePoint(sc.nc, io.ID, func() data.Point {
						p := data.NewPointFloat(data.PointTypeOffline, "", 0)
						p.Origin = sc.config.ID
						return p
					}(), false)

					if err != nil {
						log.Println("Error setting io offline:", err)
					} else {
						sc.config.IOs[i].Offline = false
					}
				}
				break
			}
			if found {
				break
			}

			newIO := ShellyIo{
				ID:         uuid.New().String(),
				DeviceID:   id,
				Parent:     sc.config.ID,
				Type:       di.model(),
				Generation: int(di.gen()),
				IP:         ip,
			}

			ne, err := data.Encode(newIO)
			if err != nil {
				log.Println("Error encoding new shelly IO:", err)
				continue
			}

			// Seed the node with a point per component the device reports, so
			// the UI has something to render and a rule has something to write
			// before the first status arrives.
			comps, err := newIO.components()
			if err != nil {
				log.Println("Error reading shelly components:", ip, err)
			}
			for _, comp := range comps {
				if state, ok := shellyCompStatePoint[comp.name]; ok {
					ne.Points = append(ne.Points, data.Point{Type: state, Key: comp.id})
				}
				if set, ok := shellyCompSetPoint[comp.name]; ok {
					ne.Points = append(ne.Points, data.Point{Type: set, Key: comp.id})
				}
			}

			err = SendNode(sc.nc, ne, sc.config.ID)
			if err != nil {
				log.Println("Error sending shelly IO:", err)
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

// A Shelly device advertises itself over mDNS with a hostname that starts with
// "shelly" and ends with the device id, such as
// "ShellyPlugUS-C049EF8889A0.local.". The hostname is used to recognize a
// Shelly and to name it; what the device is comes from the device itself.
var reShellyHost = regexp.MustCompile(`(?i)^shelly.*-([0-9a-f]+)\.local\.?$`)

// shellyHost reports whether an mDNS hostname belongs to a Shelly device.
func shellyHost(host string) bool {
	return reShellyHost.MatchString(host)
}

// shellyDeviceID returns the id that identifies a device across address
// changes. A Gen2 or later device reports its own id; for Gen1 the id is the
// serial number in the mDNS hostname.
func shellyDeviceID(host string, di shellyDeviceInfo) string {
	if di.ID != "" {
		return di.ID
	}
	m := reShellyHost.FindStringSubmatch(host)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}
