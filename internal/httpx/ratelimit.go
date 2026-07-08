package httpx

import (
	"sync"
	"time"
)

// RateLimiter is a simple fixed-window limiter keyed by an arbitrary
// string (e.g. client IP): Allow returns false once a key has hit limit
// attempts within the trailing window.
//
// ponytail: in-memory map, single process — fine for one admin login form
// at this app's scale. Entries for keys that stop trying are never
// evicted; add a periodic sweep (or an LRU) if this ever needs to survive
// a sustained distributed attack across many IPs.
type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{attempts: make(map[string][]time.Time), limit: limit, window: window}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-rl.window)
	kept := rl.attempts[key][:0]
	for _, t := range rl.attempts[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= rl.limit {
		rl.attempts[key] = kept
		return false
	}
	rl.attempts[key] = append(kept, time.Now())
	return true
}
