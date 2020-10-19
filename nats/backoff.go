package nats

import (
	"math"
	"math/rand"
	"time"
)

// ExpBackoff calculates an exponential time backup to max duration + a random fraction of 1s
func ExpBackoff(attempts int, max time.Duration) time.Duration {
	delay := max

	// if attempts is too large, then things soon start to overflow
	// so only calculate when # of attempts is relatively small
	if attempts < 30 {
		calc := time.Duration(math.Exp2(float64(attempts))) * time.Second
		// if math.Exp2(..) is +Inf, then converting that to duration
		// ends up being zero. If attempts is large, then duration may
		// be negative -- should be covered by attempts > 50 above.
		if calc > 0 || calc < max {
			delay = calc
		}
	}

	// randomize a bit
	delay = delay + time.Duration(rand.Float32()*1000)*time.Millisecond
	return delay
}
