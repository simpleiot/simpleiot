package client

import (
	"math"
	"math/rand"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

// gpsEarthRadius is the mean radius of the earth in meters, used for the
// great-circle movement calculation
const gpsEarthRadius = 6371000.0

// gpsSim generates a plausible GPS track. It moves at a configured speed on a
// heading that drifts randomly, producing a track that wanders without
// teleporting.
//
// This type deliberately has no NATS, serial, or network dependency so the
// movement math can be tested directly.
type gpsSim struct {
	lat     float64 // degrees, +N
	lon     float64 // degrees, +E
	heading float64 // degrees true, 0-360
	speed   float64 // meters/second
	// headingRate bounds how far the heading may drift, in degrees/second
	headingRate float64
	// altitude the track holds, in meters, before per-step jitter
	baseAltitude float64
	rand         *rand.Rand

	// values refreshed each step so the simulated fix looks live
	altitude float64
	numSat   int
	hdop     float64
}

func newGPSSim(config GPS, r *rand.Rand) *gpsSim {
	s := &gpsSim{
		lat:          config.SimLatitude,
		lon:          gpsNormalizeLongitude(config.SimLongitude),
		heading:      gpsNormalizeHeading(config.SimHeading),
		speed:        config.SimSpeed,
		headingRate:  config.SimHeadingRate,
		baseAltitude: 100,
		rand:         r,
	}
	s.refreshQuality()
	return s
}

// refreshQuality jitters the values a real receiver would vary between fixes
func (s *gpsSim) refreshQuality() {
	s.altitude = s.baseAltitude + (s.rand.Float64()*4 - 2)
	s.numSat = 8 + s.rand.Intn(5)
	s.hdop = 0.8 + s.rand.Float64()*0.7
}

// step advances the simulated position by dt and returns the resulting fix
func (s *gpsSim) step(dt time.Duration) gpsFix {
	secs := dt.Seconds()
	if secs > 0 {
		// drift the heading before moving, so a turn applies over this step
		s.heading = gpsNormalizeHeading(
			s.heading + (s.rand.Float64()*2-1)*s.headingRate*secs,
		)

		if d := s.speed * secs; d != 0 {
			s.lat, s.lon = gpsDestination(s.lat, s.lon, s.heading, d)
		}

		s.refreshQuality()
	}

	return s.fix()
}

// fix returns the current simulated position as a fix
func (s *gpsSim) fix() gpsFix {
	now := time.Now()
	return gpsFix{
		Latitude:  gpsPtr(s.lat),
		Longitude: gpsPtr(s.lon),
		Altitude:  gpsPtr(s.altitude),
		Speed:     gpsPtr(s.speed),
		Heading:   gpsPtr(s.heading),
		// The simulator reports a normal GPS fix rather than the "simulated"
		// fix quality both NMEA and gpsd define, so downstream logic behaves
		// exactly as it would with real hardware. The node's gpsSource point
		// is what tells an operator this data is synthetic.
		FixType:    gpsPtr(data.PointValueFix3D),
		FixQuality: gpsPtr(data.PointValueFixQualityGPS),
		NumSat:     gpsPtr(s.numSat),
		HDOP:       gpsPtr(s.hdop),
		Time:       gpsPtr(now),
	}
}

// gpsDestination returns the position reached by travelling distance meters
// from lat/lon along a great circle on the given heading.
//
// The great-circle formula is used rather than a flat-earth approximation
// because it stays correct near the poles and across the antimeridian.
// See https://www.movable-type.co.uk/scripts/latlong.html
func gpsDestination(lat, lon, heading, distance float64) (float64, float64) {
	lat1 := lat * math.Pi / 180
	lon1 := lon * math.Pi / 180
	bearing := heading * math.Pi / 180
	// angular distance travelled, in radians
	angDist := distance / gpsEarthRadius

	sinLat2 := math.Sin(lat1)*math.Cos(angDist) +
		math.Cos(lat1)*math.Sin(angDist)*math.Cos(bearing)
	// the identity bounds this to [-1, 1], but rounding can push it a hair
	// outside near the poles, and math.Asin returns NaN for that
	sinLat2 = clamp(sinLat2, -1, 1)
	lat2 := math.Asin(sinLat2)

	lon2 := lon1 + math.Atan2(
		math.Sin(bearing)*math.Sin(angDist)*math.Cos(lat1),
		math.Cos(angDist)-math.Sin(lat1)*sinLat2,
	)

	return lat2 * 180 / math.Pi, gpsNormalizeLongitude(lon2*180/math.Pi)
}

// gpsNormalizeHeading wraps a heading in degrees into [0, 360)
func gpsNormalizeHeading(heading float64) float64 {
	heading = math.Mod(heading, 360)
	if heading < 0 {
		heading += 360
	}
	if heading >= 360 {
		// possible when heading is a tiny negative value and adding 360
		// rounds up to exactly 360
		heading = 0
	}
	return heading
}

// gpsNormalizeLongitude wraps a longitude in degrees into [-180, 180)
func gpsNormalizeLongitude(lon float64) float64 {
	lon = math.Mod(lon+180, 360)
	if lon < 0 {
		lon += 360
	}
	lon -= 180
	if lon >= 180 {
		lon -= 360
	}
	return lon
}

// runSim generates a simulated GPS track and publishes it until stopped
func (gc *GPSClient) runSim(config GPS, src *gpsSource) {
	sim := newGPSSim(config, rand.New(rand.NewSource(time.Now().UnixNano())))

	conn := &gpsConnState{gc: gc}
	counters := &gpsCounters{gc: gc}

	gc.log.Printf("%v: simulating GPS from %.5f, %.5f at %v m/s",
		config.Description, config.SimLatitude, config.SimLongitude,
		config.SimSpeed)

	conn.set(true)

	// publish the starting position right away rather than making a consumer
	// wait a full period for the first fix
	gc.publish(sim.fix())
	counters.countRx()

	period := time.Duration(config.Period * float64(time.Second))
	t := time.NewTicker(period)
	defer t.Stop()

	last := time.Now()

	for {
		select {
		case now := <-t.C:
			gc.publish(sim.step(now.Sub(last)))
			counters.countRx()
			last = now

		case typ := <-src.reset:
			counters.handleReset(typ)

		case <-src.stop:
			conn.set(false)
			return
		}
	}
}
