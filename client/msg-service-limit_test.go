package client

import (
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	b := newTokenBucket(3, time.Minute, now)

	for i := 0; i < 3; i++ {
		if !b.take(now) {
			t.Fatalf("take %v refused inside burst", i)
		}
	}
	if b.take(now) {
		t.Fatal("take allowed over burst")
	}

	// half a refill interval later, still nothing
	if b.take(now.Add(30 * time.Second)) {
		t.Fatal("take allowed before refill")
	}

	// one interval after the last take, one token is back
	now = now.Add(time.Minute)
	if !b.take(now) {
		t.Fatal("take refused after refill")
	}
	if b.take(now) {
		t.Fatal("more than one token refilled")
	}

	// a long idle period refills to burst and no further
	now = now.Add(time.Hour)
	for i := 0; i < 3; i++ {
		if !b.take(now) {
			t.Fatalf("take %v refused after idle", i)
		}
	}
	if b.take(now) {
		t.Fatal("refilled past burst")
	}

	// a clock that goes backwards does not add tokens
	if b.take(now.Add(-time.Hour)) {
		t.Fatal("take allowed after clock went backwards")
	}
}
