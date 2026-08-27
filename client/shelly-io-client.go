package client

import (
	"encoding/json"
	"log"
	"math"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

const (
	// shellyPollGen1 is how often a Gen1 device is read. Gen1 answers the whole
	// device in one request, and it has no push channel here, so it is polled.
	shellyPollGen1 = time.Second * 2
	// shellyPollGen2 backs up the WebSocket for anything a device does not push
	// and re-syncs after a missed frame.
	shellyPollGen2 = time.Second * 60
	// shellyPollOffline is how often to retry a device that is not answering.
	shellyPollOffline = time.Minute * 10
	// shellySyncConfig is how often the device name is compared with the node
	// description.
	shellySyncConfig = time.Minute * 10
)

// ShellyIOClient is a SIOT client for a single Shelly device
type ShellyIOClient struct {
	nc            *nats.Conn
	config        ShellyIo
	points        data.Points
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
	errorCount    int
	comps         []shellyComp
	// status is the last status the device reported, kept so that a partial
	// NotifyStatus can be merged into it rather than replacing it.
	status  map[string]json.RawMessage
	watcher *shellyWatcher
}

// NewShellyIOClient ...
func NewShellyIOClient(nc *nats.Conn, config ShellyIo) Client {
	// we need a copy of points with timestamps so we know when to send up new data
	ne, err := data.Encode(config)
	if err != nil {
		log.Println("Error encoding shelly config:", err)
	}

	return &ShellyIOClient{
		nc:            nc,
		config:        config,
		points:        ne.Points,
		status:        map[string]json.RawMessage{},
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
	}
}

// Run runs the main logic for this client and blocks until stopped
func (sioc *ShellyIOClient) Run() error {
	log.Println("Starting shelly IO client:", sioc.config.Description)

	sioc.syncDeviceInfo()

	pushed := sioc.config.Gen() >= ShellyGen2

	sampleRate := shellyPollGen1
	if pushed {
		sampleRate = shellyPollGen2
	}

	syncConfigTicker := time.NewTicker(shellySyncConfig)
	sampleTicker := time.NewTicker(sampleRate)

	if sioc.config.Offline && !pushed {
		sampleTicker = time.NewTicker(shellyPollOffline)
	}

	if sioc.config.Disabled {
		sampleTicker.Stop()
	}

	// A Gen2 or later device pushes its status, so the connection carries both
	// the data and whether the device is reachable.
	var updates chan shellyStatusUpdate

	watch := func(on bool) {
		if !pushed {
			return
		}
		if !on {
			if sioc.watcher != nil {
				sioc.watcher.Stop()
				sioc.watcher = nil
			}
			updates = nil
			return
		}
		if sioc.watcher != nil {
			return
		}
		sioc.watcher = newShellyWatcher(sioc.config.IP)
		sioc.watcher.Start()
		updates = sioc.watcher.updates
	}

	watch(!sioc.config.Disabled)

	shellyError := func() {
		sioc.errorCount++
		if !sioc.config.Offline && sioc.errorCount > 5 {
			sioc.setOffline(true)
			if !pushed {
				sampleTicker = time.NewTicker(shellyPollOffline)
			}
		}
	}

	shellyCommOK := func() {
		sioc.errorCount = 0
		if sioc.config.Offline {
			sioc.setOffline(false)
			if !pushed {
				sampleTicker = time.NewTicker(sampleRate)
			}
		}
	}

	syncConfig := func() {
		config, err := sioc.config.getConfig()
		if err != nil {
			shellyError()
			log.Println("Error getting shelly IO settings:", sioc.config.Desc(), err)
			return
		}

		shellyCommOK()

		if sioc.config.Description == "" && config.Name != "" {
			sioc.config.Description = config.Name
			err := SendNodePoint(sioc.nc, sioc.config.ID,
				data.NewPointString(data.PointTypeDescription, "", config.Name), false)
			if err != nil {
				log.Println("Error sending shelly io description:", err)
			}
		} else if sioc.config.Description != config.Name {
			err := sioc.config.SetName(sioc.config.Description)
			if err != nil {
				log.Println("Error setting name on Shelly device:", err)
			}
		}
	}

	syncConfig()

	// Learn what the device is made of. A device that is not answering yet is
	// asked again on the next poll.
	sioc.syncComps()

	poll := func() {
		points, err := sioc.config.GetStatus()
		if err != nil {
			log.Printf("Error getting status for %v: %v\n", sioc.config.Description, err)
			shellyError()
			return
		}
		shellyCommOK()
		sioc.publish(points)
		sioc.applySets()
	}

	if !pushed {
		poll()
	}

done:
	for {
		select {
		case <-sioc.stop:
			log.Println("Stopping shelly IO client:", sioc.config.Description)
			break done

		case pts := <-sioc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &sioc.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}

			set := false
			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeDescription:
					syncConfig()
				case data.PointTypeDisabled:
					if p.Val() == 0 {
						sampleTicker = time.NewTicker(sampleRate)
						watch(true)
					} else {
						sampleTicker.Stop()
						watch(false)
					}
				case data.PointTypeSwitchSet, data.PointTypeLightSet,
					data.PointTypePositionSet:
					set = true
				}
			}

			// Act on a command as soon as it arrives rather than waiting for
			// the next tick.
			if set {
				sioc.applySets()
			}

		case pts := <-sioc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &sioc.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}

		case <-syncConfigTicker.C:
			syncConfig()

		case u := <-updates:
			if u.err != nil {
				log.Printf("Shelly device %v connection lost: %v\n",
					sioc.config.Description, u.err)
			}
			if !u.connected {
				if !sioc.config.Offline {
					sioc.setOffline(true)
				}
				break
			}

			shellyCommOK()
			if u.full {
				sioc.status = u.status
				sioc.syncCompsFromStatus()
			} else {
				shellyMergeStatus(sioc.status, u.status)
			}
			sioc.publish(shellyPointsFromStatus(u.status))
			// A device that was switched at the wall reports the change here,
			// and a controlled device is driven back to what it was told.
			sioc.applySets()

		case <-sampleTicker.C:
			if sioc.config.Disabled {
				continue
			}
			poll()
		}
	}

	// clean up
	sampleTicker.Stop()
	syncConfigTicker.Stop()
	if sioc.watcher != nil {
		sioc.watcher.Stop()
	}
	return nil
}

// setOffline records whether the device is reachable.
func (sioc *ShellyIOClient) setOffline(offline bool) {
	if offline {
		log.Printf("Shelly device %v is offline", sioc.config.Description)
	} else {
		log.Printf("Shelly device %v is online", sioc.config.Description)
	}
	sioc.config.Offline = offline
	err := SendNodePoint(sioc.nc, sioc.config.ID,
		data.NewPointFloat(data.PointTypeOffline, "", data.BoolToFloat(offline)), false)
	if err != nil {
		log.Println("ShellyIO: error sending node point:", err)
	}
}

// syncDeviceInfo fills in what the device reports about itself when the node
// does not carry it yet. A node created before the client recorded the
// generation has no way to know whether to expect the RPC API, so it asks.
func (sioc *ShellyIOClient) syncDeviceInfo() {
	if sioc.config.Generation != 0 && sioc.config.Type != "" {
		return
	}

	di, err := shellyGetDeviceInfo(sioc.config.IP)
	if err != nil {
		log.Println("Error getting shelly device info:", sioc.config.Desc(), err)
		return
	}

	pts := data.Points{}
	if sioc.config.Generation != int(di.gen()) {
		sioc.config.Generation = int(di.gen())
		pts = append(pts, data.NewPointInt(data.PointTypeGeneration, "",
			int64(sioc.config.Generation)))
	}
	if model := di.model(); model != "" && sioc.config.Type != model {
		sioc.config.Type = model
		pts = append(pts, data.NewPointString(data.PointTypeType, "", model))
	}
	if len(pts) == 0 {
		return
	}
	if err := SendNodePoints(sioc.nc, sioc.config.ID, pts, false); err != nil {
		log.Println("shelly io: error sending device info:", err)
	}
}

// syncComps asks the device what components it has.
func (sioc *ShellyIOClient) syncComps() {
	comps, err := sioc.config.components()
	if err != nil {
		log.Println("Error reading shelly components:", sioc.config.Desc(), err)
		return
	}
	sioc.comps = comps
}

// syncCompsFromStatus updates the component list from a full status, so that a
// device whose add-on module was added or removed is picked up without a
// restart.
func (sioc *ShellyIOClient) syncCompsFromStatus() {
	sioc.comps = shellyCompsFromStatus(sioc.status)
}

// applySets drives every component whose set point differs from its state.
func (sioc *ShellyIOClient) applySets() {
	if !sioc.config.Controlled {
		return
	}

	for _, comp := range sioc.comps {
		switch shellyCompSetPoint[comp.name] {
		case data.PointTypeSwitchSet:
			set, ok := sioc.config.SwitchSet[comp.id]
			if !ok || sioc.config.Switch[comp.id] == set {
				continue
			}
			if err := sioc.config.SetOnOff(comp.name, comp.id, set); err != nil {
				log.Printf("Error setting %v %v: %v\n", sioc.config.Description, comp.name, err)
			}

		case data.PointTypeLightSet:
			set, ok := sioc.config.LightSet[comp.id]
			if !ok || sioc.config.Light[comp.id] == set {
				continue
			}
			if err := sioc.config.SetOnOff(comp.name, comp.id, set); err != nil {
				log.Printf("Error setting %v %v: %v\n", sioc.config.Description, comp.name, err)
			}

		case data.PointTypePositionSet:
			set, ok := sioc.config.PositionSet[comp.id]
			// A cover stops near the position it was given rather than exactly
			// on it, so a small difference is not worth another command.
			if !ok || math.Abs(sioc.config.Position[comp.id]-set) < 1 {
				continue
			}
			if err := sioc.config.SetPosition(comp.id, set); err != nil {
				log.Printf("Error setting %v position: %v\n", sioc.config.Description, err)
			}
		}
	}
}

// publish sends on the points that changed and merges them into the local copy
// of the config.
func (sioc *ShellyIOClient) publish(points data.Points) {
	newPoints := sioc.points.Merge(points, time.Minute*15)
	if len(newPoints) <= 0 {
		return
	}

	err := data.MergePoints(sioc.config.ID, newPoints, &sioc.config)
	if err != nil {
		log.Println("shelly io: error merging newPoints:", err)
	}
	err = SendNodePoints(sioc.nc, sioc.config.ID, newPoints, false)
	if err != nil {
		log.Println("shelly io: error sending newPoints:", err)
	}
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
