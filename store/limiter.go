package store

import (
	"sync"
	"time"
)

// Sign-in attempts are limited per account. After authLimitFailures
// failures in a row an account is refused for authLimitBase, doubling with
// each further failure up to authLimitMax; a successful sign-in clears
// the count. The limiter keys on what the caller presents (an email
// address, or a user ID on the NATS side), so a guess against one account
// does not lock another, and it forgets an account that has been quiet
// for authLimitForget.
const (
	authLimitFailures = 5
	authLimitBase     = time.Second
	authLimitMax      = 5 * time.Minute
	authLimitForget   = 15 * time.Minute
)

type authEntry struct {
	failures int
	until    time.Time
	last     time.Time
}

// authLimiter is an in-memory sign-in limiter shared by every entry point
// that checks a password or a token.
type authLimiter struct {
	mu      sync.Mutex
	entries map[string]*authEntry
	now     func() time.Time
}

func newAuthLimiter() *authLimiter {
	return &authLimiter{entries: make(map[string]*authEntry), now: time.Now}
}

// allowed reports whether an attempt for key may be checked now.
func (l *authLimiter) allowed(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.forget(now)

	e, ok := l.entries[key]
	if !ok {
		return true
	}
	return !now.Before(e.until)
}

// failed records a failed attempt for key and returns how long further
// attempts are refused, zero when they are not yet.
func (l *authLimiter) failed(key string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	e, ok := l.entries[key]
	if !ok {
		e = &authEntry{}
		l.entries[key] = e
	}
	e.failures++
	e.last = now

	over := e.failures - authLimitFailures
	if over < 0 {
		return 0
	}
	delay := authLimitBase << uint(min(over, 16))
	if delay > authLimitMax {
		delay = authLimitMax
	}
	e.until = now.Add(delay)
	return delay
}

// succeeded clears the record for key.
func (l *authLimiter) succeeded(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

// forget drops entries that have been quiet; called with the lock held.
func (l *authLimiter) forget(now time.Time) {
	for k, e := range l.entries {
		if now.Sub(e.last) > authLimitForget && !now.Before(e.until) {
			delete(l.entries, k)
		}
	}
}
