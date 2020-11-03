package nats

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func TestBackoffFirstAttempt(t *testing.T) {
	fmt.Println("exp: ", math.Exp2(1))
	backoff := ExpBackoff(1, time.Second*6)
	if backoff < time.Second*2 || backoff > 3*time.Second {
		t.Error("backoff time is out of range: ", backoff)

	}
}

func TestBackoffLargeAttempts(t *testing.T) {
	backoff := ExpBackoff(400000, time.Second)

	if backoff < time.Second {
		t.Error("backoff time is too short: ", backoff)
	}

	if backoff > 2*time.Second {
		t.Error("backoff time is too long: ", backoff)
	}

}

func TestBackoff16(t *testing.T) {
	backoff := ExpBackoff(16, time.Minute*6)
	if backoff < time.Minute*6 || backoff > time.Minute*6+time.Second {
		t.Error("backoff should be 6m, was: ", backoff)
	}
}
