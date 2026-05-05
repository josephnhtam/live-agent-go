package transcriber

import (
	"context"
	"time"
)

func backoff(ctx context.Context, failures int, base, max time.Duration) bool {
	duration := base
	if failures > 1 {
		duration = base * time.Duration(1<<(failures-1))
	}

	if duration > max {
		duration = max
	}

	select {
	case <-time.After(duration):
		return true
	case <-ctx.Done():
		return false
	}
}
