package utils

import (
    "sync"
    "time"
)

type TokenBucket struct {
	capacity int
	tokens   int
	rate     time.Duration
	mu       sync.Mutex
}

func NewTokenBucket(capacity int, rate time.Duration) *TokenBucket {
	return &TokenBucket{
		capacity: capacity,
		tokens:   capacity,
		rate:     rate,
	}
}

func (tb *TokenBucket) Take() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	return false
}

func (tb *TokenBucket) Refill() {
	for {
		time.Sleep(tb.rate)
		tb.mu.Lock()
		if tb.tokens < tb.capacity {
			tb.tokens++
		}
		tb.mu.Unlock()
	}
}