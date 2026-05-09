package helper

import "time"

type TokenBucket struct {
	interval  time.Duration
	maxTokens float64
	tokens    float64
	lastTick  time.Time
}

func NewTokenBucket(interval time.Duration, maxTokens int) *TokenBucket {
	return &TokenBucket{
		interval:  interval,
		maxTokens: float64(maxTokens),
	}
}

func (b *TokenBucket) Take() {
	now := time.Now()

	if !b.lastTick.IsZero() {
		elapsed := now.Sub(b.lastTick)
		gained := float64(elapsed) / float64(b.interval)

		b.tokens += gained
		if b.tokens > b.maxTokens {
			b.tokens = b.maxTokens
		}
	} else {
		b.tokens = b.maxTokens
	}

	b.lastTick = now

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return
	}

	waitFor := time.Duration((1.0 - b.tokens) * float64(b.interval))
	if waitFor > 0 {
		time.Sleep(waitFor)
	}

	b.tokens = 0.0
	b.lastTick = time.Now()
}
