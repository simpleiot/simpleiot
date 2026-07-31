package client

import (
	"log"
	"os"
	"sync"
	"sync/atomic"
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
	SimReset       bool    `point:"simReset"`
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

// gpsPosition is the most recently published position. A simulator starts from
// here rather than from its configured start position, so restarting a client
// or the application continues the track instead of returning to the
// beginning.
//
// It is written by the running source goroutine through publish and read by
// the Run loop when a source starts, so access is guarded.
type gpsPosition struct {
	mu      sync.Mutex
	valid   bool
	lat     float64
	lon     float64
	heading float64
}

// set records a position. The heading is left unchanged when the source does
// not report one, which receivers commonly omit while stationary.
func (p *gpsPosition) set(lat, lon float64, heading *float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.valid = true
	p.lat = lat
	p.lon = lon
	if heading != nil {
		p.heading = *heading
	}
}

// seed initializes the position from the node's stored points so a client
// started with the application picks up where the last run left off. A node
// that has never had a fix is left unset, which starts a simulator from its
// configured start position.
func (p *gpsPosition) seed(config GPS) {
	if config.FixType == data.PointValueFixNone {
		return
	}
	p.set(config.Latitude, config.Longitude, &config.Heading)
}

// clear forgets the last position, sending the next simulator back to its
// configured start position
func (p *gpsPosition) clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.valid = false
}

// get returns the last position and whether one has been recorded
func (p *gpsPosition) get() (lat, lon, heading float64, valid bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lat, p.lon, p.heading, p.valid
}

// GPSClient is a SIOT client that reads GPS position data
type GPSClient struct {
	log           *log.Logger
	nc            *nats.Conn
	config        GPS
	nodeID        string
	pos           gpsPosition
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
}

// NewGPSClient returns a new GPSClient using its configuration read from the
// Client Manager
func NewGPSClient(nc *nats.Conn, config GPS) Client {
	return &GPSClient{
		log:    log.New(os.Stderr, "gps: ", log.LstdFlags|log.Lmsgprefix),
		nc:     nc,
		config: config,
		// nodeID is captured separately because config is owned by the Run
		// loop and must not be read from a source goroutine
		nodeID:        config.ID,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
	}
}

// gpsSource carries control signals from the client Run loop to the running
// source goroutine.
//
// A source goroutine must never read or write GPSClient.config -- the Run loop
// owns it and mutates it on every incoming point. Everything a source needs is
// either passed in by value at start, kept local to the goroutine, or reached
// through this type.
type gpsSource struct {
	// stop is closed to shut the source down. Closing rather than sending
	// means shutdown never blocks, even if the source has already returned.
	stop chan struct{}
	// reset carries point types whose counters should be zeroed
	reset chan string
	// debug is the level the source logs at. It lives here rather than in the
	// config a source is started with so a change applies to the running
	// source: restarting one to change how much it logs would drop a serial
	// connection or a gpsd session. It outlives a gpsd reconnect, which starts
	// a session with a fresh copy of the config.
	debug atomic.Int64
}

func newGPSSource(debug int) *gpsSource {
	src := &gpsSource{
		stop:  make(chan struct{}),
		reset: make(chan string, 4),
	}
	src.setDebug(debug)
	return src
}

// debugLevel returns the level the source should currently log at
func (s *gpsSource) debugLevel() int {
	return int(s.debug.Load())
}

// setDebug changes the level a running source logs at
func (s *gpsSource) setDebug(debug int) {
	s.debug.Store(int64(debug))
}

// gpsCounters tracks the receive and error counts for a source goroutine.
// Kept local to the goroutine rather than in GPSClient.config to avoid racing
// with the Run loop.
type gpsCounters struct {
	gc         *GPSClient
	rx         int
	errorCount int
}

func (c *gpsCounters) countRx() {
	c.rx++
	c.gc.sendStatus(data.PointTypeRx, float64(c.rx))
}

func (c *gpsCounters) countError() {
	c.errorCount++
	c.gc.sendStatus(data.PointTypeErrorCount, float64(c.errorCount))
}

// handleReset zeroes the counter named by a reset request from the Run loop
func (c *gpsCounters) handleReset(typ string) {
	switch typ {
	case data.PointTypeRx:
		c.rx = 0
		c.gc.sendStatus(data.PointTypeRx, 0)
	case data.PointTypeErrorCount:
		c.errorCount = 0
		c.gc.sendStatus(data.PointTypeErrorCount, 0)
	}
}

// gpsConnState publishes the connected point when the state changes. Like
// gpsCounters, it is local to a source goroutine.
type gpsConnState struct {
	gc        *GPSClient
	connected bool
	known     bool
}

func (c *gpsConnState) set(connected bool) {
	if c.known && c.connected == connected {
		return
	}
	c.known = true
	c.connected = connected
	c.gc.sendStatus(data.PointTypeConnected, data.BoolToFloat(connected))
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

// resumeSim returns config with the simulator start position replaced by the
// last published position, so a restarted simulator continues its track. The
// configured start position is kept when there is no position to resume from.
func (gc *GPSClient) resumeSim(config GPS) GPS {
	lat, lon, heading, valid := gc.pos.get()
	if !valid {
		return config
	}

	config.SimLatitude = lat
	config.SimLongitude = lon
	config.SimHeading = heading
	return config
}

// publish sends a fix to the GPS node. Every point in the fix shares one
// timestamp.
//
// The points sent are returned so a source can log exactly what it published,
// which is how the simulator provides the detail the hardware sources get by
// logging their raw input. A fix carrying no values sends nothing and returns
// nothing.
func (gc *GPSClient) publish(fix gpsFix) data.Points {
	pts := fix.points(time.Now())
	if len(pts) <= 0 {
		return nil
	}

	if fix.Latitude != nil && fix.Longitude != nil {
		gc.pos.set(*fix.Latitude, *fix.Longitude, fix.Heading)
	}

	err := SendNodePoints(gc.nc, gc.nodeID, pts, false)
	if err != nil {
		gc.log.Printf("Error sending points: %v", err)
	}

	return pts
}

// sendStatus publishes a single status point on the GPS node
func (gc *GPSClient) sendStatus(typ string, value float64) {
	err := SendNodePoints(gc.nc, gc.nodeID,
		data.Points{data.NewPointFloat(typ, "", value)}, false)
	if err != nil {
		gc.log.Printf("Error sending %v point: %v", typ, err)
	}
}

// runSource dispatches to the configured source and blocks until the source
// is stopped
func (gc *GPSClient) runSource(config GPS, src *gpsSource) {
	if config.Disabled {
		gc.log.Printf("%v: disabled", config.Description)
		return
	}

	switch config.Source {
	case data.PointValueGPSSourceSim:
		gc.runSim(config, src)
	case data.PointValueGPSSourceGpsd:
		gc.runGpsd(config, src)
	default:
		gc.runSerial(config, src)
	}
}

// gpsSimStartPoints are the config changes that move the simulated receiver to
// the configured start position. Every other restart resumes the track from
// the last published position.
var gpsSimStartPoints = map[string]bool{
	data.PointTypeSimLatitude:  true,
	data.PointTypeSimLongitude: true,
	data.PointTypeSimHeading:   true,
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

	var src *gpsSource

	// pick up the position the node was last at, so a simulator continues its
	// track across an application restart
	gc.pos.seed(gc.config)

	// startSource starts the configured source. A simulator resumes from the
	// last published position unless startPos is set, which sends it back to
	// the configured start position.
	startSource := func(startPos bool) {
		config := gc.config.withDefaults()
		if startPos {
			gc.pos.clear()
		} else {
			config = gc.resumeSim(config)
		}

		src = newGPSSource(config.Debug)
		go gc.runSource(config, src)
	}

	stopCurrentSource := func() {
		if src != nil {
			close(src.stop)
			src = nil
		}
	}

	// requestReset asks the running source to zero a counter. The send is
	// non-blocking so a busy source can never stall the Run loop; dropping a
	// reset is preferable to blocking every other client message behind it.
	requestReset := func(typ string) {
		if src == nil {
			return
		}
		select {
		case src.reset <- typ:
		default:
			gc.log.Printf("%v: reset request dropped, source busy",
				gc.config.Description)
		}
	}

	startSource(false)

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
			startPos := false
			for _, p := range pts.Points {
				if gpsRestartPoints[p.Type] {
					restart = true
				}

				// editing the start position moves the simulated receiver
				// there rather than continuing the previous track
				if gpsSimStartPoints[p.Type] {
					startPos = true
				}

				switch p.Type {
				case data.PointTypeRxReset:
					if p.Bool() {
						requestReset(data.PointTypeRx)
					}
				case data.PointTypeErrorCountReset:
					if p.Bool() {
						requestReset(data.PointTypeErrorCount)
					}
				case data.PointTypeDebug:
					// applied to the running source rather than restarting
					// it. The merge above has already converted the point, so
					// the level is read back from the config.
					if src != nil {
						src.setDebug(gc.config.Debug)
					}
				case data.PointTypeSimReset:
					if p.Bool() {
						restart = true
						startPos = true

						// clear the request so the reset can be repeated
						gc.config.SimReset = false
						gc.sendStatus(data.PointTypeSimReset, 0)
					}
				}
			}

			// restart once even when several config points arrive together
			if restart {
				stopCurrentSource()
				startSource(startPos)
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
