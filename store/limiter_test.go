package store

import (
	"testing"
	"time"
)

func TestAuthLimiter(t *testing.T) {
	now := time.Unix(1000, 0)
	l := newAuthLimiter()
	l.now = func() time.Time { return now }

	for i := 0; i < authLimitFailures; i++ {
		if !l.allowed("a") {
			t.Fatalf("refused after %v failures", i)
		}
		if d := l.failed("a"); i < authLimitFailures-1 && d != 0 {
			t.Fatalf("delay %v after %v failures", d, i+1)
		}
	}
	// the fifth failure starts the lockout
	if l.allowed("a") {
		t.Fatal("allowed during lockout")
	}
	if !l.allowed("b") {
		t.Fatal("another account was locked")
	}
	now = now.Add(authLimitBase)
	if !l.allowed("a") {
		t.Fatal("still refused after the delay")
	}
	// each further failure doubles the delay, up to the cap
	if d := l.failed("a"); d != 2*authLimitBase {
		t.Fatalf("second delay %v", d)
	}
	for range 20 {
		l.failed("a")
	}
	now = now.Add(authLimitMax - time.Second)
	if l.allowed("a") {
		t.Fatal("delay was not capped from below")
	}
	now = now.Add(2 * time.Second)
	if !l.allowed("a") {
		t.Fatal("delay exceeded the cap")
	}
	// success clears it
	l.succeeded("a")
	if d := l.failed("a"); d != 0 {
		t.Fatalf("failure count survived success: %v", d)
	}
	// a quiet account is forgotten
	now = now.Add(authLimitForget + time.Second)
	l.allowed("x")
	if _, ok := l.entries["a"]; ok {
		t.Fatal("quiet entry kept")
	}
}
