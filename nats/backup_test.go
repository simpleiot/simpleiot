package nats

import (
	"fmt"
	"testing"
	"time"
)

func TestBackupLargeAttempts(t *testing.T) {
	backoff := ExpBackoff(400000, time.Second)

	fmt.Println("backoff: ", backoff)

	if backoff < time.Second {
		t.Error("backoff time is too short: ", backoff)
	}

	if backoff > 2*time.Second {
		t.Error("backoff time is too long: ", backoff)
	}

}
