package core

import (
	"sync/atomic"
	"time"
)

type RateLimiter struct {
	rate           int
	burst          int
	tokens         atomic.Int64
	lastRefill     atomic.Int64
	overflowAction string
}

func NewRateLimiter(rate, burst int, overflowAction string) *RateLimiter {
	rl := &RateLimiter{
		rate:           rate,
		burst:          burst,
		overflowAction: overflowAction,
	}
	rl.tokens.Store(int64(burst))
	rl.lastRefill.Store(time.Now().UnixMilli())
	return rl
}

func (rl *RateLimiter) Allow() bool {
	rl.refill()

	tokens := rl.tokens.Load()
	if tokens > 0 {
		rl.tokens.Add(-1)
		return true
	}

	switch rl.overflowAction {
	case "aggregate":
		rl.tokens.Add(-1)
		return true
	case "drop":
		return false
	case "warn":
		return false
	default:
		return false
	}
}

func (rl *RateLimiter) refill() {
	now := time.Now().UnixMilli()
	last := rl.lastRefill.Load()

	elapsed := now - last
	if elapsed <= 0 {
		return
	}

	tokensToAdd := (int64(elapsed) * int64(rl.rate)) / 1000
	if tokensToAdd <= 0 {
		return
	}

	current := rl.tokens.Load()
	newTokens := current + tokensToAdd
	if newTokens > int64(rl.burst) {
		newTokens = int64(rl.burst)
	}

	rl.tokens.Store(newTokens)
	rl.lastRefill.Store(now)
}

func (rl *RateLimiter) tokensToAdd(elapsedMs int64) int64 {
	return (elapsedMs * int64(rl.rate)) / 1000
}

func (rl *RateLimiter) GetTokens() int64 {
	rl.refill()
	return rl.tokens.Load()
}

func (rl *RateLimiter) GetBurst() int {
	return rl.burst
}

func (rl *RateLimiter) GetRate() int {
	return rl.rate
}

func (rl *RateLimiter) GetOverflowAction() string {
	return rl.overflowAction
}

func (rl *RateLimiter) TryWait(n int) bool {
	for i := 0; i < n; i++ {
		if !rl.Allow() {
			return false
		}
	}
	return true
}
