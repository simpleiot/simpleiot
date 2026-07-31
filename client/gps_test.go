package client

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

// gpsTestSentence wraps an NMEA sentence body in the leading $ and the
// trailing checksum, so tests can write sentences without computing checksums
// by hand
func gpsTestSentence(body string) string {
	sum := byte(0)
	for i := 0; i < len(body); i++ {
		sum ^= body[i]
	}
	return fmt.Sprintf("$%s*%02X", body, sum)
}

// newTestGPSSim returns a simulator with a deterministic random source so
// tests do not depend on wall-clock seeding
func newTestGPSSim(config GPS) *gpsSim {
	return newGPSSim(config, rand.New(rand.NewSource(1)))
}

func TestGPSNormalizeHeading(t *testing.T) {
	tests := []struct {
		in, exp float64
	}{
		{0, 0},
		{90, 90},
		{359.9, 359.9},
		{360, 0},
		{450, 90},
		{-10, 350},
		{-360, 0},
		{-370, 350},
		{720, 0},
	}

	for _, test := range tests {
		got := gpsNormalizeHeading(test.in)
		if math.Abs(got-test.exp) > 1e-9 {
			t.Errorf("heading %v: expected %v, got %v", test.in, test.exp, got)
		}
		if got < 0 || got >= 360 {
			t.Errorf("heading %v: result %v out of range [0, 360)", test.in, got)
		}
	}
}

func TestGPSNormalizeLongitude(t *testing.T) {
	tests := []struct {
		in, exp float64
	}{
		{0, 0},
		{179.9, 179.9},
		{180, -180},
		{-180, -180},
		{181, -179},
		{-181, 179},
		{360, 0},
		{540, -180},
	}

	for _, test := range tests {
		got := gpsNormalizeLongitude(test.in)
		if math.Abs(got-test.exp) > 1e-9 {
			t.Errorf("longitude %v: expected %v, got %v", test.in, test.exp, got)
		}
		if got < -180 || got >= 180 {
			t.Errorf("longitude %v: result %v out of range [-180, 180)",
				test.in, got)
		}
	}
}

// TestGPSDestinationDueNorth checks the great-circle movement against a known
// distance. One degree of latitude is a fixed arc length regardless of
// longitude, so this is exact for a spherical earth.
func TestGPSDestinationDueNorth(t *testing.T) {
	// arc length of one degree of latitude on a sphere of gpsEarthRadius
	oneDegree := gpsEarthRadius * math.Pi / 180

	lat, lon := gpsDestination(40, -75, 0, oneDegree)

	if math.Abs(lat-41) > 1e-6 {
		t.Errorf("expected latitude 41, got %v", lat)
	}
	if math.Abs(lon-(-75)) > 1e-6 {
		t.Errorf("expected longitude to be unchanged at -75, got %v", lon)
	}
}

// TestGPSDestinationDueEast checks movement along the equator, where a degree
// of longitude has the same arc length as a degree of latitude
func TestGPSDestinationDueEast(t *testing.T) {
	oneDegree := gpsEarthRadius * math.Pi / 180

	lat, lon := gpsDestination(0, 10, 90, oneDegree)

	if math.Abs(lat) > 1e-6 {
		t.Errorf("expected latitude to stay at 0, got %v", lat)
	}
	if math.Abs(lon-11) > 1e-6 {
		t.Errorf("expected longitude 11, got %v", lon)
	}
}

// TestGPSDestinationCrossAntimeridian verifies the longitude wraps rather than
// running off past 180, which a flat-earth approximation would get wrong
func TestGPSDestinationCrossAntimeridian(t *testing.T) {
	// start just west of the antimeridian and travel east past it
	oneDegree := gpsEarthRadius * math.Pi / 180

	lat, lon := gpsDestination(0, 179.5, 90, oneDegree)

	if math.Abs(lat) > 1e-6 {
		t.Errorf("expected latitude to stay at 0, got %v", lat)
	}
	if math.Abs(lon-(-179.5)) > 1e-6 {
		t.Errorf("expected longitude to wrap to -179.5, got %v", lon)
	}
}

// TestGPSSimStraightTrack verifies a zero heading rate produces a straight
// track, and that the distance covered matches the configured speed
func TestGPSSimStraightTrack(t *testing.T) {
	sim := newTestGPSSim(GPS{
		SimLatitude:    40,
		SimLongitude:   -75,
		SimHeading:     0,
		SimSpeed:       100,
		SimHeadingRate: 0,
	})

	for i := 0; i < 10; i++ {
		sim.step(time.Second)
		if sim.heading != 0 {
			t.Fatalf("step %v: heading drifted to %v with a zero heading rate",
				i, sim.heading)
		}
		if math.Abs(sim.lon-(-75)) > 1e-9 {
			t.Fatalf("step %v: longitude drifted to %v travelling due north",
				i, sim.lon)
		}
	}

	// 10 steps of 1 second at 100 m/s is 1000 m due north
	expected := 40 + (1000/gpsEarthRadius)*180/math.Pi
	if math.Abs(sim.lat-expected) > 1e-6 {
		t.Errorf("expected latitude %v after 1000 m north, got %v",
			expected, sim.lat)
	}
}

// TestGPSSimBoundedTurn verifies the heading drifts but never by more than the
// configured rate allows
func TestGPSSimBoundedTurn(t *testing.T) {
	const headingRate = 5.0

	sim := newTestGPSSim(GPS{
		SimLatitude:    40,
		SimLongitude:   -75,
		SimHeading:     90,
		SimSpeed:       10,
		SimHeadingRate: headingRate,
	})

	moved := false
	last := sim.heading

	for i := 0; i < 200; i++ {
		sim.step(time.Second)

		// shortest angular distance between the two headings
		delta := math.Abs(math.Mod(sim.heading-last+540, 360) - 180)
		if delta > headingRate+1e-9 {
			t.Fatalf("step %v: heading changed %v degrees, limit is %v",
				i, delta, headingRate)
		}
		if delta > 0 {
			moved = true
		}
		last = sim.heading
	}

	if !moved {
		t.Error("heading never changed with a nonzero heading rate")
	}
}

// TestGPSSimStaysInRange runs a long track from an awkward starting point and
// verifies the coordinates never leave their valid ranges
func TestGPSSimStaysInRange(t *testing.T) {
	starts := []struct {
		name     string
		lat, lon float64
	}{
		{"antimeridian", 0, 179.9},
		{"north pole", 89.9, 0},
		{"south pole", -89.9, 0},
	}

	for _, start := range starts {
		t.Run(start.name, func(t *testing.T) {
			sim := newTestGPSSim(GPS{
				SimLatitude:    start.lat,
				SimLongitude:   start.lon,
				SimHeading:     0,
				SimSpeed:       500,
				SimHeadingRate: 45,
			})

			for i := 0; i < 5000; i++ {
				sim.step(time.Second)

				if sim.lat < -90 || sim.lat > 90 || math.IsNaN(sim.lat) {
					t.Fatalf("step %v: latitude %v out of range", i, sim.lat)
				}
				if sim.lon < -180 || sim.lon >= 180 || math.IsNaN(sim.lon) {
					t.Fatalf("step %v: longitude %v out of range", i, sim.lon)
				}
				if sim.heading < 0 || sim.heading >= 360 {
					t.Fatalf("step %v: heading %v out of range", i, sim.heading)
				}
			}
		})
	}
}

// TestGPSFixSharedTimestamp is the check that keeps a Grafana geomap working.
// Latitude and longitude must land on the same timestamp or they cannot be
// joined into one row.
func TestGPSFixSharedTimestamp(t *testing.T) {
	sim := newTestGPSSim(GPS{
		SimLatitude:    40,
		SimLongitude:   -75,
		SimSpeed:       10,
		SimHeadingRate: 5,
	})

	now := time.Now()
	pts := sim.step(time.Second).points(now)

	if len(pts) < 2 {
		t.Fatalf("expected a full fix, got %v points", len(pts))
	}

	for _, p := range pts {
		if !p.Time.Equal(now) {
			t.Errorf("point %v has time %v, expected %v", p.Type, p.Time, now)
		}
	}
}

// TestGPSSimFixContents verifies the simulator reports a complete fix, so
// downstream consumers see the same shape they would from real hardware
func TestGPSSimFixContents(t *testing.T) {
	sim := newTestGPSSim(GPS{
		SimLatitude:    40,
		SimLongitude:   -75,
		SimSpeed:       10,
		SimHeadingRate: 5,
	})

	fix := sim.step(time.Second)

	if fix.Latitude == nil || fix.Longitude == nil || fix.Altitude == nil ||
		fix.Speed == nil || fix.Heading == nil || fix.Time == nil {
		t.Fatal("simulated fix is missing position fields")
	}

	if fix.FixType == nil || *fix.FixType != 3 {
		t.Errorf("expected a 3D fix type, got %v", fix.FixType)
	}
	if fix.FixQuality == nil || *fix.FixQuality != 1 {
		t.Errorf("expected a GPS fix quality, got %v", fix.FixQuality)
	}
	if fix.NumSat == nil || *fix.NumSat < 8 || *fix.NumSat > 12 {
		t.Errorf("expected 8-12 satellites, got %v", fix.NumSat)
	}
	if fix.HDOP == nil || *fix.HDOP < 0.8 || *fix.HDOP > 1.5 {
		t.Errorf("expected an HDOP of 0.8-1.5, got %v", fix.HDOP)
	}
}

// TestGPSFixOmitsAbsentFields verifies a partial fix publishes only what it
// has, rather than filling in zeros
func TestGPSFixOmitsAbsentFields(t *testing.T) {
	fix := gpsFix{
		Latitude:  gpsPtr(40.0),
		Longitude: gpsPtr(-75.0),
	}

	pts := fix.points(time.Now())

	if len(pts) != 2 {
		t.Fatalf("expected 2 points from a position-only fix, got %v: %v",
			len(pts), pts)
	}
}

// TestGPSFixZeroValuesPublished verifies that zero is treated as a real value
// rather than an absent one -- 0,0 is a position and a speed of 0 means
// stationary
func TestGPSFixZeroValuesPublished(t *testing.T) {
	fix := gpsFix{
		Latitude:  gpsPtr(0.0),
		Longitude: gpsPtr(0.0),
		Speed:     gpsPtr(0.0),
	}

	pts := fix.points(time.Now())

	if len(pts) != 3 {
		t.Fatalf("expected 3 points, got %v: %v", len(pts), pts)
	}
}

// gpsFeed pushes sentences through an accumulator and returns the last
// completed fix, or nil if no cycle completed
func gpsFeed(t *testing.T, a *gpsNMEAAccumulator, sentences ...string) *gpsFix {
	t.Helper()

	var last *gpsFix
	for _, s := range sentences {
		fix, err := a.add(s)
		if err != nil {
			t.Fatalf("unexpected error parsing %q: %v", s, err)
		}
		if fix != nil {
			last = fix
		}
	}
	return last
}

func TestGPSNMEAParseGGA(t *testing.T) {
	a := newGPSNMEAAccumulator()

	// a GGA with a 3 satellite fix off the coast of Sydney
	gga := gpsTestSentence(
		"GPGGA,034225.077,3356.4650,S,15124.5567,E,1,03,9.7,-25.0,M,21.0,M,,0000")

	// the second GGA closes the first cycle
	fix := gpsFeed(t, a, gga, gga)
	if fix == nil {
		t.Fatal("expected a completed fix after a repeated sentence type")
	}

	if fix.Latitude == nil || math.Abs(*fix.Latitude-(-33.9410833)) > 1e-6 {
		t.Errorf("expected latitude -33.9410833, got %v", fix.Latitude)
	}
	if fix.Longitude == nil || math.Abs(*fix.Longitude-151.4092783) > 1e-6 {
		t.Errorf("expected longitude 151.4092783, got %v", fix.Longitude)
	}
	if fix.Altitude == nil || math.Abs(*fix.Altitude-(-25.0)) > 1e-9 {
		t.Errorf("expected altitude -25, got %v", fix.Altitude)
	}
	if fix.FixQuality == nil || *fix.FixQuality != 1 {
		t.Errorf("expected fix quality 1, got %v", fix.FixQuality)
	}
	if fix.NumSat == nil || *fix.NumSat != 3 {
		t.Errorf("expected 3 satellites, got %v", fix.NumSat)
	}
	if fix.HDOP == nil || math.Abs(*fix.HDOP-9.7) > 1e-9 {
		t.Errorf("expected HDOP 9.7, got %v", fix.HDOP)
	}
}

func TestGPSNMEAParseRMC(t *testing.T) {
	a := newGPSNMEAAccumulator()

	rmc := gpsTestSentence(
		"GNRMC,220516,A,5133.82,N,00042.24,W,173.8,231.8,130694,004.2,W")

	fix := gpsFeed(t, a, rmc, rmc)
	if fix == nil {
		t.Fatal("expected a completed fix")
	}

	// NMEA reports speed in knots, SIOT publishes meters/second
	expSpeed := 173.8 * gpsKnotsToMPS
	if fix.Speed == nil || math.Abs(*fix.Speed-expSpeed) > 1e-6 {
		t.Errorf("expected speed %v m/s, got %v", expSpeed, fix.Speed)
	}
	if fix.Heading == nil || math.Abs(*fix.Heading-231.8) > 1e-9 {
		t.Errorf("expected heading 231.8, got %v", fix.Heading)
	}
	if fix.Latitude == nil || math.Abs(*fix.Latitude-51.5636667) > 1e-6 {
		t.Errorf("expected latitude 51.5636667, got %v", fix.Latitude)
	}
	if fix.Longitude == nil || math.Abs(*fix.Longitude-(-0.704)) > 1e-6 {
		t.Errorf("expected longitude -0.704, got %v", fix.Longitude)
	}

	// the two digit year 94 windows back to 1994, not 2094
	if fix.Time == nil {
		t.Fatal("expected a receiver timestamp from RMC")
	}
	exp := time.Date(1994, time.June, 13, 22, 5, 16, 0, time.UTC)
	if !fix.Time.Equal(exp) {
		t.Errorf("expected time %v, got %v", exp, *fix.Time)
	}
}

func TestGPSNMEAParseGSA(t *testing.T) {
	a := newGPSNMEAAccumulator()

	gsa := gpsTestSentence("GNGSA,A,3,80,71,73,79,69,,,,,,,,1.83,1.09,1.47")

	fix := gpsFeed(t, a, gsa, gsa)
	if fix == nil {
		t.Fatal("expected a completed fix")
	}

	if fix.FixType == nil || *fix.FixType != 3 {
		t.Errorf("expected a 3D fix type, got %v", fix.FixType)
	}
}

// TestGPSNMEAGSANoFixMapsToZero covers the one place the NMEA and SIOT fix
// type encodings differ: NMEA numbers a missing fix 1, SIOT and gpsd use 0
func TestGPSNMEAGSANoFixMapsToZero(t *testing.T) {
	a := newGPSNMEAAccumulator()

	gsa := gpsTestSentence("GPGSA,A,1,,,,,,,,,,,,,,,")

	fix := gpsFeed(t, a, gsa, gsa)
	if fix == nil {
		t.Fatal("expected a completed fix")
	}

	if fix.FixType == nil || *fix.FixType != 0 {
		t.Errorf("expected fix type 0 for a NMEA fix type of 1, got %v",
			fix.FixType)
	}
}

func TestGPSNMEAParseVTG(t *testing.T) {
	a := newGPSNMEAAccumulator()

	vtg := gpsTestSentence("GPVTG,45.5,T,67.5,M,30.45,N,56.40,K")

	fix := gpsFeed(t, a, vtg, vtg)
	if fix == nil {
		t.Fatal("expected a completed fix")
	}

	expSpeed := 30.45 * gpsKnotsToMPS
	if fix.Speed == nil || math.Abs(*fix.Speed-expSpeed) > 1e-6 {
		t.Errorf("expected speed %v m/s, got %v", expSpeed, fix.Speed)
	}
	if fix.Heading == nil || math.Abs(*fix.Heading-45.5) > 1e-9 {
		t.Errorf("expected heading 45.5, got %v", fix.Heading)
	}
}

// TestGPSNMEARMCPrefersRMCOverVTG verifies VTG only fills in speed and heading
// when RMC has not already supplied them
func TestGPSNMEARMCPrefersRMCOverVTG(t *testing.T) {
	a := newGPSNMEAAccumulator()

	rmc := gpsTestSentence(
		"GNRMC,220516,A,5133.82,N,00042.24,W,10.0,90.0,130694,004.2,W")
	vtg := gpsTestSentence("GPVTG,45.5,T,67.5,M,30.45,N,56.40,K")

	fix := gpsFeed(t, a, rmc, vtg, rmc)
	if fix == nil {
		t.Fatal("expected a completed fix")
	}

	expSpeed := 10.0 * gpsKnotsToMPS
	if fix.Speed == nil || math.Abs(*fix.Speed-expSpeed) > 1e-6 {
		t.Errorf("expected RMC speed %v m/s to win over VTG, got %v",
			expSpeed, fix.Speed)
	}
	if fix.Heading == nil || math.Abs(*fix.Heading-90.0) > 1e-9 {
		t.Errorf("expected RMC heading 90 to win over VTG, got %v", fix.Heading)
	}
}

// TestGPSNMEAInvalidRMCRejected verifies a receiver reporting a stale position
// with validity V does not publish that position as a fix
func TestGPSNMEAInvalidRMCRejected(t *testing.T) {
	a := newGPSNMEAAccumulator()

	rmc := gpsTestSentence(
		"GNRMC,220516,V,5133.82,N,00042.24,W,173.8,231.8,130694,004.2,W")

	fix := gpsFeed(t, a, rmc, rmc)
	if fix == nil {
		t.Fatal("expected a completed fix")
	}

	if fix.Latitude != nil || fix.Longitude != nil {
		t.Errorf("expected no position from an invalid RMC, got %v, %v",
			fix.Latitude, fix.Longitude)
	}
	if fix.FixQuality == nil || *fix.FixQuality != 0 {
		t.Errorf("expected fix quality 0 from an invalid RMC, got %v",
			fix.FixQuality)
	}
}

// TestGPSNMEANoFixNotAnError covers a cold receiver. It emits well-formed
// sentences with empty position fields, which go-nmea rejects; that must be
// reported as no fix rather than counted as a data error.
func TestGPSNMEANoFixNotAnError(t *testing.T) {
	a := newGPSNMEAAccumulator()

	gga := gpsTestSentence("GPGGA,123519,,,,,0,00,,,M,,M,,")

	if _, err := a.add(gga); err != nil {
		t.Fatalf("a no-fix GGA should not be a parse error, got: %v", err)
	}

	fix, err := a.add(gga)
	if err != nil {
		t.Fatalf("a no-fix GGA should not be a parse error, got: %v", err)
	}
	if fix == nil {
		t.Fatal("expected a completed fix")
	}

	if fix.FixQuality == nil || *fix.FixQuality != 0 {
		t.Errorf("expected fix quality 0 with no fix, got %v", fix.FixQuality)
	}
	if fix.Latitude != nil || fix.Longitude != nil {
		t.Errorf("expected no position with no fix, got %v, %v",
			fix.Latitude, fix.Longitude)
	}
}

// TestGPSNMEACorruptSentence verifies genuinely bad data is reported as an
// error, unlike the no-fix case above
func TestGPSNMEACorruptSentence(t *testing.T) {
	a := newGPSNMEAAccumulator()

	tests := []struct {
		name string
		line string
	}{
		{
			name: "bad checksum",
			line: "$GPGGA,034225.077,3356.4650,S,15124.5567,E,1,03,9.7,-25.0,M,21.0,M,,0000*FF",
		},
		{
			name: "not a sentence",
			line: "this is not NMEA data",
		},
		{
			name: "truncated",
			line: "$GPGG",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := a.add(test.line); err == nil {
				t.Errorf("expected an error for %q", test.line)
			}
		})
	}
}

// TestGPSNMEACycleMerge verifies one publish per NMEA cycle carrying the
// fields from every sentence in that cycle, rather than a partial fix per
// sentence
func TestGPSNMEACycleMerge(t *testing.T) {
	a := newGPSNMEAAccumulator()

	gga := gpsTestSentence(
		"GPGGA,034225.077,3356.4650,S,15124.5567,E,1,03,9.7,-25.0,M,21.0,M,,0000")
	gsa := gpsTestSentence("GNGSA,A,3,80,71,73,79,69,,,,,,,,1.83,1.09,1.47")
	rmc := gpsTestSentence(
		"GNRMC,220516,A,3356.4650,S,15124.5567,E,10.0,90.0,130694,004.2,W")

	// no fix should complete until a type repeats
	for _, s := range []string{gga, gsa, rmc} {
		fix, err := a.add(s)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fix != nil {
			t.Fatalf("fix completed early on %q", s)
		}
	}

	fix, err := a.add(gga)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fix == nil {
		t.Fatal("expected the cycle to complete on the repeated GGA")
	}

	// altitude and fix quality come from GGA, fix type from GSA, speed and
	// heading from RMC -- all in one fix
	if fix.Altitude == nil {
		t.Error("expected altitude from GGA")
	}
	if fix.FixQuality == nil || *fix.FixQuality != 1 {
		t.Errorf("expected fix quality 1 from GGA, got %v", fix.FixQuality)
	}
	if fix.FixType == nil || *fix.FixType != 3 {
		t.Errorf("expected fix type 3 from GSA, got %v", fix.FixType)
	}
	if fix.Speed == nil {
		t.Error("expected speed from RMC")
	}
	if fix.Heading == nil {
		t.Error("expected heading from RMC")
	}

	// and every point in that merged fix shares one timestamp
	now := time.Now()
	for _, p := range fix.points(now) {
		if !p.Time.Equal(now) {
			t.Errorf("point %v has time %v, expected %v", p.Type, p.Time, now)
		}
	}
}

// TestGPSNMEAIgnoresUnconsumedSentences verifies sentences the client does not
// use, such as the several GSV sentences a receiver sends per cycle, neither
// error nor falsely trigger cycle detection
func TestGPSNMEAIgnoresUnconsumedSentences(t *testing.T) {
	a := newGPSNMEAAccumulator()

	gga := gpsTestSentence(
		"GPGGA,034225.077,3356.4650,S,15124.5567,E,1,03,9.7,-25.0,M,21.0,M,,0000")
	gsv1 := gpsTestSentence("GPGSV,3,1,11,03,03,111,00,04,15,270,00,06,01,010,00,13,06,292,00")
	gsv2 := gpsTestSentence("GPGSV,3,2,11,14,25,170,00,16,57,208,39,18,67,296,40,19,40,246,00")

	for _, s := range []string{gga, gsv1, gsv2} {
		fix, err := a.add(s)
		if err != nil {
			t.Fatalf("unexpected error on %q: %v", s, err)
		}
		if fix != nil {
			t.Fatalf("repeated GSV sentences falsely completed a cycle")
		}
	}

	fix, err := a.add(gga)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fix == nil {
		t.Fatal("expected the cycle to complete on the repeated GGA")
	}
}
