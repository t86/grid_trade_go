package limit

import (
	"errors"
	"sync"
	"time"
)

var ErrRateLimited = errors.New("rate limited")

type TokenBucket struct {
	mu       sync.Mutex
	capacity int
	tokens   int
	refillAt time.Time
	interval time.Duration
}

func NewTokenBucket(capacity int, interval time.Duration) *TokenBucket {
	return &TokenBucket{
		capacity: capacity,
		tokens:   capacity,
		refillAt: time.Now().Add(interval),
		interval: interval,
	}
}

func (b *TokenBucket) Allow(_ string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if time.Now().After(b.refillAt) {
		b.tokens = b.capacity
		b.refillAt = time.Now().Add(b.interval)
	}
	if b.tokens == 0 {
		return ErrRateLimited
	}

	b.tokens--
	return nil
}
