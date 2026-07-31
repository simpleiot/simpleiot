package client

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

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
