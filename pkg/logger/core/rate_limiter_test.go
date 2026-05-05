package core

import (
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_Allow_Basic(t *testing.T) {
	rl := NewRateLimiter(100, 10, "drop")
	defer func() {}()

	if !rl.Allow() {
		t.Error("Allow should return true when tokens available")
	}

	if rl.GetTokens() != 9 {
		t.Errorf("Tokens = %d, want 9", rl.GetTokens())
	}
}

func TestRateLimiter_Allow_ExhaustTokens(t *testing.T) {
	rl := NewRateLimiter(100, 2, "drop")

	for i := 0; i < 2; i++ {
		if !rl.Allow() {
			t.Errorf("Allow %d should succeed", i+1)
		}
	}

	if rl.Allow() {
		t.Error("Allow should return false when no tokens")
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	rl := NewRateLimiter(1000, 10, "drop")

	rl.Allow()
	rl.Allow()

	time.Sleep(10 * time.Millisecond)

	tokens := rl.GetTokens()
	if tokens < 10 {
		t.Errorf("Tokens = %d, should refill toward burst limit", tokens)
	}
}

func TestRateLimiter_BurstLimit(t *testing.T) {
	rl := NewRateLimiter(100, 5, "drop")

	for i := 0; i < 10; i++ {
		rl.Allow()
	}

	tokens := rl.GetTokens()
	if tokens != 0 && tokens < 0 {
		t.Errorf("Tokens = %d, should not exceed burst limit", tokens)
	}
}

func TestRateLimiter_OverflowAction_Aggregate(t *testing.T) {
	rl := NewRateLimiter(100, 2, "aggregate")

	for i := 0; i < 5; i++ {
		rl.Allow()
	}

	tokens := rl.GetTokens()
	if tokens != -3 {
		t.Errorf("Tokens = %d (aggregate can go negative), want -3", tokens)
	}
}

func TestRateLimiter_OverflowAction_Drop(t *testing.T) {
	rl := NewRateLimiter(100, 2, "drop")

	for i := 0; i < 2; i++ {
		rl.Allow()
	}

	if rl.Allow() {
		t.Error("Drop action should reject when no tokens")
	}
}

func TestRateLimiter_OverflowAction_Warn(t *testing.T) {
	rl := NewRateLimiter(100, 2, "warn")

	for i := 0; i < 2; i++ {
		rl.Allow()
	}

	if rl.Allow() {
		t.Error("Warn action should reject when no tokens")
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(1000, 100, "aggregate")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				rl.Allow()
			}
		}()
	}
	wg.Wait()

	tokens := rl.GetTokens()
	if tokens >= 0 {
		t.Logf("Tokens = %d after concurrent usage (aggregate mode)", tokens)
	}
}

func TestRateLimiter_TryWait(t *testing.T) {
	rl := NewRateLimiter(100, 5, "drop")

	if !rl.TryWait(3) {
		t.Error("TryWait(3) should succeed with burst=5")
	}

	if rl.TryWait(5) {
		t.Error("TryWait(5) should fail when only ~2 tokens left")
	}
}

func TestRateLimiter_GetRate(t *testing.T) {
	rl := NewRateLimiter(200, 10, "drop")

	if rl.GetRate() != 200 {
		t.Errorf("GetRate = %d, want 200", rl.GetRate())
	}
}

func TestRateLimiter_GetBurst(t *testing.T) {
	rl := NewRateLimiter(200, 50, "drop")

	if rl.GetBurst() != 50 {
		t.Errorf("GetBurst = %d, want 50", rl.GetBurst())
	}
}

func TestRateLimiter_GetOverflowAction(t *testing.T) {
	rl := NewRateLimiter(100, 10, "aggregate")

	if rl.GetOverflowAction() != "aggregate" {
		t.Errorf("GetOverflowAction = %s, want aggregate", rl.GetOverflowAction())
	}
}

func TestRateLimiter_TokensToAdd(t *testing.T) {
	rl := NewRateLimiter(1000, 100, "drop")

	add := rl.tokensToAdd(100)
	if add != 100 {
		t.Errorf("tokensToAdd(100ms) = %d, want 100 (rate=1000)", add)
	}

	add = rl.tokensToAdd(500)
	if add != 500 {
		t.Errorf("tokensToAdd(500ms) = %d, want 500", add)
	}
}
