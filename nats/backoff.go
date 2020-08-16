package nats

import (
	"math"
	"math/rand"
	"time"
)

// ExpBackoff calculates an exponential time backup to max duration + a random fraction of 1s
func ExpBackoff(attempts int, max time.Duration) time.Duration {
	delay := time.Duration(math.Exp2(float64(attempts))) * time.Second
	if delay > max {
		delay = max
	}
	// randomize a bit
	delay = delay + time.Duration(rand.Float32()*1000)*time.Millisecond
	return delay
}
