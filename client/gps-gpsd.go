package client

import (
	"encoding/json"
	"net"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

const (
	// gpsdDialTimeout bounds how long a connection attempt waits
	gpsdDialTimeout = 10 * time.Second

	// gpsdStaleFixDur is how long the client waits for a TPV report before
	// declaring itself disconnected. gpsd holds the TCP connection open when
	// the receiver goes quiet or is unplugged, so a live socket is not
	// evidence of a live fix.
	gpsdStaleFixDur = 10 * time.Second

	// gpsdMaxBackoff caps the reconnect delay
	gpsdMaxBackoff = time.Minute
)

// gpsd report classes the client consumes
const (
	gpsdClassTPV     = "TPV"
	gpsdClassSKY     = "SKY"
	gpsdClassVersion = "VERSION"
)

// gpsdReport is used to read the class tag before decoding the full object
type gpsdReport struct {
	Class string `json:"class"`
}

// gpsdTPV is a gpsd time-position-velocity report.
//
// Numeric fields are pointers because gpsd omits any value it does not have,
// and zero is meaningful for most of them: a speed of 0 means stationary, and
// a latitude and longitude of 0 is a real position in the Gulf of Guinea.
// Decoding into a value type would turn "gpsd said nothing" into "gpsd said
// zero".
type gpsdTPV struct {
	Device string   `json:"device"`
	Mode   *int     `json:"mode"`
	Status *int     `json:"status"`
	Time   string   `json:"time"`
	Lat    *float64 `json:"lat"`
	Lon    *float64 `json:"lon"`
	Alt    *float64 `json:"alt"`
	AltHAE *float64 `json:"altHAE"`
	AltMSL *float64 `json:"altMSL"`
	Speed  *float64 `json:"speed"`
	Track  *float64 `json:"track"`
}

// gpsdSKY is a gpsd satellite and dilution-of-precision report
type gpsdSKY struct {
	Device     string   `json:"device"`
	HDOP       *float64 `json:"hdop"`
	NSat       *int     `json:"nSat"`
	USat       *int     `json:"uSat"`
	Satellites []struct {
		Used bool `json:"used"`
	} `json:"satellites"`
}

// gpsdVersion is sent by gpsd when a client connects
type gpsdVersion struct {
	Release string `json:"release"`
	Rev     string `json:"rev"`
}

// gpsd TPV status values, which describe the augmentation quality of a fix.
// These are gpsd's own numbering and differ from the NMEA GGA numbering SIOT
// publishes, so they are mapped rather than passed through.
const (
	gpsdStatusUnknown     = 0
	gpsdStatusNormal      = 1
	gpsdStatusDGPS        = 2
	gpsdStatusRTKFixed    = 3
	gpsdStatusRTKFloat    = 4
	gpsdStatusDR          = 5
	gpsdStatusGNSSDR      = 6
	gpsdStatusTimeSurveys = 7
	gpsdStatusSimulated   = 8
	gpsdStatusPY          = 9
)

// gpsdFixType converts a gpsd TPV mode to the SIOT fix type.
//
// SIOT adopts gpsd's encoding, so this is nearly a pass-through; the one
// difference is that gpsd distinguishes "unknown" (0) from "no fix" (1) where
// SIOT uses 0 for both.
func gpsdFixType(mode *int) (int, bool) {
	if mode == nil {
		return 0, false
	}

	switch *mode {
	case 2:
		return data.PointValueFix2D, true
	case 3:
		return data.PointValueFix3D, true
	default:
		return data.PointValueFixNone, true
	}
}

// gpsdFixQuality converts a gpsd TPV status to the SIOT fix quality, which
// follows the NMEA GGA numbering.
//
// status is optional; when it is absent the quality is inferred from mode,
// since a 2D or 3D fix means the receiver has a usable position.
func gpsdFixQuality(status, mode *int) (int, bool) {
	haveFix := mode != nil && (*mode == 2 || *mode == 3)

	if status == nil {
		if mode == nil {
			return 0, false
		}
		if haveFix {
			return data.PointValueFixQualityGPS, true
		}
		return data.PointValueFixQualityNone, true
	}

	switch *status {
	case gpsdStatusNormal:
		return data.PointValueFixQualityGPS, true
	case gpsdStatusDGPS:
		return data.PointValueFixQualityDGPS, true
	case gpsdStatusRTKFixed:
		return data.PointValueFixQualityRTKFixed, true
	case gpsdStatusRTKFloat:
		return data.PointValueFixQualityRTKFloat, true
	case gpsdStatusDR, gpsdStatusGNSSDR:
		return data.PointValueFixQualityEstimated, true
	case gpsdStatusSimulated:
		return data.PointValueFixQualitySimulated, true
	case gpsdStatusPY:
		// NMEA's PPS means Precise Positioning Service, the military precise
		// code, which is what gpsd's P(Y) describes
		return data.PointValueFixQualityPPS, true
	case gpsdStatusUnknown, gpsdStatusTimeSurveys:
		// no SIOT equivalent; report a plain GPS fix when the position is
		// usable, since the position is what consumers act on
		if haveFix {
			return data.PointValueFixQualityGPS, true
		}
		return data.PointValueFixQualityNone, true
	default:
		if haveFix {
			return data.PointValueFixQualityGPS, true
		}
		return data.PointValueFixQualityNone, true
	}
}

// gpsdAltitude picks the best available altitude from a TPV report.
//
// gpsd deprecated the plain alt field because it was ambiguous about which
// datum it used, replacing it with altMSL (mean sea level) and altHAE (height
// above the WGS84 ellipsoid). Older gpsd builds send only alt. Preferring
// altMSL keeps the value consistent with the serial source, which reports the
// mean sea level altitude from GGA.
func gpsdAltitude(tpv *gpsdTPV) *float64 {
	switch {
	case tpv.AltMSL != nil:
		return tpv.AltMSL
	case tpv.AltHAE != nil:
		return tpv.AltHAE
	default:
		return tpv.Alt
	}
}

// decodeGpsd merges one gpsd report into fix. It returns true when the report
// completes a position and the fix should be published.
func decodeGpsd(msg json.RawMessage, fix *gpsFix) (bool, error) {
	var report gpsdReport
	if err := json.Unmarshal(msg, &report); err != nil {
		return false, err
	}

	switch report.Class {
	case gpsdClassTPV:
		var tpv gpsdTPV
		if err := json.Unmarshal(msg, &tpv); err != nil {
			return false, err
		}
		applyGpsdTPV(&tpv, fix)
		return true, nil

	case gpsdClassSKY:
		var sky gpsdSKY
		if err := json.Unmarshal(msg, &sky); err != nil {
			return false, err
		}
		applyGpsdSKY(&sky, fix)
		// SKY carries no position, so it updates the accumulating fix and
		// waits for the next TPV to publish
		return false, nil

	default:
		// VERSION, DEVICES, WATCH, and anything else are not errors
		return false, nil
	}
}

// applyGpsdTPV merges a position report into the accumulating fix
func applyGpsdTPV(tpv *gpsdTPV, fix *gpsFix) {
	if tpv.Lat != nil {
		fix.Latitude = gpsPtr(*tpv.Lat)
	}
	if tpv.Lon != nil {
		fix.Longitude = gpsPtr(*tpv.Lon)
	}
	if alt := gpsdAltitude(tpv); alt != nil {
		fix.Altitude = gpsPtr(*alt)
	}
	if tpv.Speed != nil {
		// gpsd already reports meters/second, unlike the NMEA knots
		fix.Speed = gpsPtr(*tpv.Speed)
	}
	if tpv.Track != nil {
		fix.Heading = gpsPtr(gpsNormalizeHeading(*tpv.Track))
	}
	if v, ok := gpsdFixType(tpv.Mode); ok {
		fix.FixType = gpsPtr(v)
	}
	if v, ok := gpsdFixQuality(tpv.Status, tpv.Mode); ok {
		fix.FixQuality = gpsPtr(v)
	}
	if tpv.Time != "" {
		if t, err := time.Parse(time.RFC3339, tpv.Time); err == nil {
			fix.Time = gpsPtr(t)
		}
	}
}

// applyGpsdSKY merges a satellite report into the accumulating fix
func applyGpsdSKY(sky *gpsdSKY, fix *gpsFix) {
	if sky.HDOP != nil {
		fix.HDOP = gpsPtr(*sky.HDOP)
	}

	switch {
	case sky.USat != nil:
		fix.NumSat = gpsPtr(*sky.USat)
	case len(sky.Satellites) > 0:
		// older gpsd versions omit uSat, so count the satellites marked used
		used := 0
		for _, sat := range sky.Satellites {
			if sat.Used {
				used++
			}
		}
		fix.NumSat = gpsPtr(used)
	}
}

// gpsdWatchCommand builds the command that starts a JSON report stream,
// optionally restricted to one device when gpsd manages several
func gpsdWatchCommand(device string) ([]byte, error) {
	watch := struct {
		Enable bool   `json:"enable"`
		JSON   bool   `json:"json"`
		Device string `json:"device,omitempty"`
	}{
		Enable: true,
		JSON:   true,
		Device: device,
	}

	b, err := json.Marshal(watch)
	if err != nil {
		return nil, err
	}

	return append(append([]byte("?WATCH="), b...), '\n'), nil
}

// runGpsd streams position reports from the gpsd daemon until stopped,
// reconnecting with a backoff whenever the connection is lost
func (gc *GPSClient) runGpsd(config GPS, src *gpsSource) {
	conn := &gpsConnState{gc: gc}
	counters := &gpsCounters{gc: gc}

	attempts := 0

	for {
		select {
		case <-src.stop:
			return
		default:
		}

		stopped, err := gc.gpsdSession(config, src, conn, counters)
		if stopped {
			return
		}

		conn.set(false)

		if err != nil {
			gc.log.Printf("%v: gpsd connection to %v ended: %v",
				config.Description, config.GpsdAddress, err)
		}

		attempts++
		delay := ExpBackoff(attempts, gpsdMaxBackoff)

		select {
		case <-time.After(delay):
		case <-src.stop:
			return
		}
	}
}

// gpsdSession runs one gpsd connection to completion. It returns stopped=true
// when the client is shutting down, and otherwise returns the error that ended
// the connection so the caller can reconnect.
func (gc *GPSClient) gpsdSession(config GPS, src *gpsSource,
	conn *gpsConnState, counters *gpsCounters) (bool, error) {

	dialer := net.Dialer{Timeout: gpsdDialTimeout}
	netConn, err := dialer.Dial("tcp", config.GpsdAddress)
	if err != nil {
		return false, err
	}
	defer netConn.Close()

	watch, err := gpsdWatchCommand(config.Device)
	if err != nil {
		return false, err
	}

	if _, err := netConn.Write(watch); err != nil {
		return false, err
	}

	gc.log.Printf("%v: connected to gpsd at %v",
		config.Description, config.GpsdAddress)

	msgs := make(chan json.RawMessage, 16)
	readErrors := make(chan error, 1)
	quit := make(chan struct{})
	defer close(quit)

	go gpsdReader(netConn, msgs, readErrors, quit)

	// gpsd keeps the socket open when the receiver stops reporting, so track
	// staleness rather than trusting the connection
	stale := time.NewTimer(gpsdStaleFixDur)
	defer stale.Stop()

	var fix gpsFix

	for {
		select {
		case msg := <-msgs:
			if src.debugLevel() >= 4 {
				gc.log.Printf("%v: %s", config.Description, msg)
			}

			counters.countRx()

			if src.debugLevel() >= 2 {
				gc.logGpsdVersion(config, msg)
			}

			publish, err := decodeGpsd(msg, &fix)
			if err != nil {
				if src.debugLevel() >= 2 {
					gc.log.Printf("%v: error decoding gpsd report: %v",
						config.Description, err)
				}
				counters.countError()
				continue
			}

			if publish {
				conn.set(true)
				gc.publish(fix)

				if !stale.Stop() {
					select {
					case <-stale.C:
					default:
					}
				}
				stale.Reset(gpsdStaleFixDur)
			}

		case err := <-readErrors:
			return false, err

		case <-stale.C:
			// still connected to gpsd, but no position is coming through
			gc.log.Printf("%v: no gpsd position report in %v",
				config.Description, gpsdStaleFixDur)
			conn.set(false)
			stale.Reset(gpsdStaleFixDur)

		case typ := <-src.reset:
			counters.handleReset(typ)

		case <-src.stop:
			return true, nil
		}
	}
}

// logGpsdVersion logs the gpsd release on connect, which is useful when
// diagnosing field differences between gpsd versions
func (gc *GPSClient) logGpsdVersion(config GPS, msg json.RawMessage) {
	var report gpsdReport
	if err := json.Unmarshal(msg, &report); err != nil {
		return
	}
	if report.Class != gpsdClassVersion {
		return
	}

	var version gpsdVersion
	if err := json.Unmarshal(msg, &version); err != nil {
		return
	}

	gc.log.Printf("%v: gpsd release %v (%v)",
		config.Description, version.Release, version.Rev)
}

// gpsdReader decodes the JSON object stream from gpsd. It runs in its own
// goroutine because reads block.
func gpsdReader(r net.Conn, msgs chan<- json.RawMessage,
	readErrors chan<- error, quit <-chan struct{}) {

	dec := json.NewDecoder(r)

	for {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			select {
			case readErrors <- err:
			case <-quit:
			}
			return
		}

		select {
		case msgs <- raw:
		case <-quit:
			return
		}
	}
}
