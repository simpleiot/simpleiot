package client

import (
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// GPS client configuration. A GPS node reads position data from one of three
// sources -- a serial NMEA receiver, the gpsd daemon, or an internal simulator
// -- and publishes the same set of points regardless of which source is used.
type GPS struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Disabled    bool   `point:"disabled"`
	Source      string `point:"gpsSource"`
	Debug       int    `point:"debug"`

	// serial source
	Port string `point:"port"`
	Baud string `point:"baud"`

	// gpsd source
	GpsdAddress string `point:"gpsdAddress"`
	Device      string `point:"device"`

	// simulation source
	SimLatitude    float64 `point:"simLatitude"`
	SimLongitude   float64 `point:"simLongitude"`
	SimSpeed       float64 `point:"simSpeed"`
	SimHeading     float64 `point:"simHeading"`
	SimHeadingRate float64 `point:"simHeadingRate"`
	Period         float64 `point:"period"`

	// status / output
	Latitude        float64 `point:"latitude"`
	Longitude       float64 `point:"longitude"`
	Altitude        float64 `point:"altitude"`
	Speed           float64 `point:"speed"`
	Heading         float64 `point:"heading"`
	FixType         int     `point:"fixType"`
	FixQuality      int     `point:"fixQuality"`
	GPSTime         float64 `point:"gpsTime"`
	NumSat          int     `point:"numSat"`
	HDOP            float64 `point:"hdop"`
	Connected       bool    `point:"connected"`
	Rx              int     `point:"rx"`
	RxReset         bool    `point:"rxReset"`
	ErrorCount      int     `point:"errorCount"`
	ErrorCountReset bool    `point:"errorCountReset"`
}

// GPS client defaults, applied when the corresponding point is not set
const (
	gpsDefaultBaud        = "9600"
	gpsDefaultGpsdAddress = "localhost:2947"
	gpsDefaultPeriod      = 1
	gpsDefaultSimSpeed    = 10
	gpsDefaultHeadingRate = 5
)

// gpsFix is the internal representation produced by every GPS source and
// consumed by the single publish path. Fields are pointers because a source
// may legitimately omit any of them, and zero is a meaningful value for most
// -- a speed of 0 means stationary, and 0,0 is a real position.
type gpsFix struct {
	Latitude   *float64
	Longitude  *float64
	Altitude   *float64
	Speed      *float64
	Heading    *float64
	FixType    *int
	FixQuality *int
	NumSat     *int
	HDOP       *float64
	// Time reported by the receiver, which may differ from system time
	Time *time.Time
}

// gpsPtr returns a pointer to v, for populating optional gpsFix fields
func gpsPtr[T any](v T) *T {
	return &v
}

// points converts a fix to points, stamping every one with the same time.
//
// The shared timestamp matters: client.SendPoints stamps each point with its
// own time.Now() when Point.Time is zero, which would leave latitude and
// longitude microseconds apart. Grafana's geomap needs both coordinates in the
// same row, and joining series with unequal timestamps yields rows where one
// coordinate is null.
func (f gpsFix) points(now time.Time) data.Points {
	pts := make(data.Points, 0, 9)

	addFloat := func(typ string, v *float64) {
		if v == nil {
			return
		}
		p := data.NewPointFloat(typ, "", *v)
		p.Time = now
		pts = append(pts, p)
	}

	addInt := func(typ string, v *int) {
		if v == nil {
			return
		}
		p := data.NewPointInt(typ, "", int64(*v))
		p.Time = now
		pts = append(pts, p)
	}

	addFloat(data.PointTypeLatitude, f.Latitude)
	addFloat(data.PointTypeLongitude, f.Longitude)
	addFloat(data.PointTypeAltitude, f.Altitude)
	addFloat(data.PointTypeSpeed, f.Speed)
	addFloat(data.PointTypeHeading, f.Heading)
	addInt(data.PointTypeFixType, f.FixType)
	addInt(data.PointTypeFixQuality, f.FixQuality)
	addInt(data.PointTypeNumSat, f.NumSat)
	addFloat(data.PointTypeHDOP, f.HDOP)

	if f.Time != nil {
		// published as Unix epoch seconds rather than an ISO 8601 string so
		// the value survives metrics-only databases
		addFloat(data.PointTypeGPSTime,
			gpsPtr(float64(f.Time.UnixNano())/float64(time.Second)))
	}

	return pts
}

// GPSClient is a SIOT client that reads GPS position data
type GPSClient struct {
	log           *log.Logger
	nc            *nats.Conn
	config        GPS
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
}

// NewGPSClient returns a new GPSClient using its configuration read from the
// Client Manager
func NewGPSClient(nc *nats.Conn, config GPS) Client {
	return &GPSClient{
		log:           log.New(os.Stderr, "gps: ", log.LstdFlags|log.Lmsgprefix),
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
	}
}

// withDefaults returns a copy of the config with unset fields filled in.
// Defaults are applied locally rather than written back as points so that
// starting a client never generates node updates.
func (g GPS) withDefaults() GPS {
	if g.Source == "" {
		g.Source = data.PointValueGPSSourceSerial
	}
	if g.Baud == "" {
		g.Baud = gpsDefaultBaud
	}
	if g.GpsdAddress == "" {
		g.GpsdAddress = gpsDefaultGpsdAddress
	}
	if g.Period <= 0 {
		g.Period = gpsDefaultPeriod
	}
	if g.SimSpeed == 0 {
		g.SimSpeed = gpsDefaultSimSpeed
	}
	if g.SimHeadingRate == 0 {
		g.SimHeadingRate = gpsDefaultHeadingRate
	}
	return g
}

// publish sends a fix to the GPS node. Every point in the fix shares one
// timestamp.
func (gc *GPSClient) publish(fix gpsFix) {
	pts := fix.points(time.Now())
	if len(pts) <= 0 {
		return
	}

	err := SendNodePoints(gc.nc, gc.config.ID, pts, false)
	if err != nil {
		gc.log.Printf("Error sending points: %v", err)
	}
}

// sendStatus publishes a single status point on the GPS node
func (gc *GPSClient) sendStatus(typ string, value float64) {
	err := SendNodePoints(gc.nc, gc.config.ID,
		data.Points{data.NewPointFloat(typ, "", value)}, false)
	if err != nil {
		gc.log.Printf("Error sending %v point: %v", typ, err)
	}
}

// setConnected publishes the connected state when it changes
func (gc *GPSClient) setConnected(connected bool) {
	if gc.config.Connected == connected {
		return
	}
	gc.config.Connected = connected
	gc.sendStatus(data.PointTypeConnected, data.BoolToFloat(connected))
}

// runSource dispatches to the configured source and blocks until stop is
// closed
func (gc *GPSClient) runSource(config GPS, stop chan struct{}) {
	if config.Disabled {
		gc.log.Printf("%v: disabled", config.Description)
		return
	}

	switch config.Source {
	case data.PointValueGPSSourceSim:
		gc.runSim(config, stop)
	case data.PointValueGPSSourceGpsd:
		gc.runGpsd(config, stop)
	default:
		gc.runSerial(config, stop)
	}
}

// gpsRestartPoints are the point types that require the source to be
// restarted. Points not listed here are either applied live or are outputs
// generated by the client itself.
var gpsRestartPoints = map[string]bool{
	data.PointTypeGPSSource:      true,
	data.PointTypeDisabled:       true,
	data.PointTypePort:           true,
	data.PointTypeBaud:           true,
	data.PointTypeGpsdAddress:    true,
	data.PointTypeDevice:         true,
	data.PointTypePeriod:         true,
	data.PointTypeSimLatitude:    true,
	data.PointTypeSimLongitude:   true,
	data.PointTypeSimSpeed:       true,
	data.PointTypeSimHeading:     true,
	data.PointTypeSimHeadingRate: true,
}

// Run the main logic for this client and blocks until stopped
func (gc *GPSClient) Run() error {
	gc.log.Printf("Starting client: %v", gc.config.Description)

	var stopSource chan struct{}

	startSource := func() {
		stopSource = make(chan struct{})
		go gc.runSource(gc.config.withDefaults(), stopSource)
	}

	// stopping the source by closing the channel rather than sending on it
	// means this never blocks, even when the source goroutine has already
	// returned
	stopCurrentSource := func() {
		if stopSource != nil {
			close(stopSource)
			stopSource = nil
		}
	}

	startSource()

done:
	for {
		select {
		case <-gc.stop:
			stopCurrentSource()
			gc.log.Printf("Stopped client: %v", gc.config.Description)
			break done

		case pts := <-gc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &gc.config)
			if err != nil {
				gc.log.Printf("Error merging new points: %v", err)
			}

			restart := false
			for _, p := range pts.Points {
				if gpsRestartPoints[p.Type] {
					restart = true
				}

				switch p.Type {
				case data.PointTypeRxReset:
					if p.Bool() {
						gc.config.Rx = 0
						gc.sendStatus(data.PointTypeRx, 0)
					}
				case data.PointTypeErrorCountReset:
					if p.Bool() {
						gc.config.ErrorCount = 0
						gc.sendStatus(data.PointTypeErrorCount, 0)
					}
				}
			}

			// restart once even when several config points arrive together
			if restart {
				stopCurrentSource()
				startSource()
			}

		case pts := <-gc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &gc.config)
			if err != nil {
				gc.log.Printf("Error merging new edge points: %v", err)
			}
		}
	}

	return nil
}

// Stop sends a signal to the Run function to exit
func (gc *GPSClient) Stop(_ error) {
	close(gc.stop)
}

// Points is called by the Manager when new points for this node are received
func (gc *GPSClient) Points(nodeID string, points []data.Point) {
	gc.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this node are
// received
func (gc *GPSClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	gc.newEdgePoints <- NewPoints{nodeID, parentID, points}
}
