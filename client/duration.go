package client

import (
	"math"
	"time"
)

// maxPointDuration is the longest period a point can ask for. It bounds every
// ticker, timer, and sleep that is built from a point value, so a value that
// would overflow cannot stall a client for years or wrap into the past.
const maxPointDuration = 30 * 24 * time.Hour

// pointDuration converts v, a period read from a point in units of unit (for
// a pollPeriod in milliseconds, time.Millisecond), to a duration that is safe
// to hand to time.NewTicker, Ticker.Reset, or time.Sleep.
//
// Points are written by users and replicated from other instances, so v is
// untrusted: a zero or negative value makes NewTicker panic and stops the
// whole instance, and a very large one overflows the multiplication into a
// duration of either sign. The result is bounded:
//
//   - v <= 0 or NaN returns def, the client's default for an unset period.
//     def may be 0 when the client treats "no period" as "do not poll".
//   - anything shorter than floor returns floor, so a tiny value cannot pin
//     the CPU.
//   - anything longer than maxPointDuration returns maxPointDuration.
func pointDuration(v float64, unit, def, floor time.Duration) time.Duration {
	if v <= 0 || math.IsNaN(v) {
		return def
	}

	f := v * float64(unit)
	if f >= float64(maxPointDuration) {
		return maxPointDuration
	}

	d := time.Duration(f)
	if d < floor {
		return floor
	}

	return d
}
