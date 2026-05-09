package helper

import (
	"context"
	"time"
)

type RetrierConfig struct {
	RetryOn                func(error) bool
	MaxConsecutiveAttempts int
	Backoff                time.Duration
	MaxBackoff             time.Duration
}

type Retrier struct {
	config RetrierConfig
}

func NewRetrier(config RetrierConfig) *Retrier {
	return &Retrier{config: config}
}

func (r *Retrier) Execute(ctx context.Context, do func(context.Context) error) error {
	consecutiveFailures := 0

	for {
		err := do(ctx)

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err == nil {
			consecutiveFailures = 0
			if err := backoff(ctx, 0, r.config.Backoff, r.config.MaxBackoff); err != nil {
				return err
			}
			continue
		}

		if r.config.RetryOn != nil && !r.config.RetryOn(err) {
			return err
		}

		consecutiveFailures++

		if r.config.MaxConsecutiveAttempts > 0 && consecutiveFailures >= r.config.MaxConsecutiveAttempts {
			return err
		}

		if err := backoff(ctx, consecutiveFailures, r.config.Backoff, r.config.MaxBackoff); err != nil {
			return err
		}
	}
}

func backoff(ctx context.Context, failures int, base, max time.Duration) error {
	duration := base
	if failures > 1 {
		duration = base * time.Duration(1<<(failures-1))
	}

	if duration > max {
		duration = max
	}

	select {
	case <-time.After(duration):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
