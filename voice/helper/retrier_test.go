package helper

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetrier_ContextCanceled(t *testing.T) {
	r := NewRetrier(RetrierConfig{
		Backoff:    time.Millisecond,
		MaxBackoff: time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := r.Execute(ctx, func(ctx context.Context) error {
		calls++
		return nil
	})
	assert.ErrorIs(t, err, context.Canceled)
	assert.Greater(t, calls, 0)
}

func TestRetrier_MaxConsecutiveAttempts(t *testing.T) {
	retryErr := errors.New("fail")
	r := NewRetrier(RetrierConfig{
		MaxConsecutiveAttempts: 3,
		Backoff:               time.Millisecond,
		MaxBackoff:            time.Millisecond,
	})

	calls := 0
	err := r.Execute(context.Background(), func(ctx context.Context) error {
		calls++
		return retryErr
	})

	assert.ErrorIs(t, err, retryErr)
	assert.Equal(t, 3, calls)
}

func TestRetrier_RetryOnFalse(t *testing.T) {
	retryErr := errors.New("not retryable")
	r := NewRetrier(RetrierConfig{
		RetryOn:    func(err error) bool { return false },
		Backoff:    time.Millisecond,
		MaxBackoff: time.Millisecond,
	})

	calls := 0
	err := r.Execute(context.Background(), func(ctx context.Context) error {
		calls++
		return retryErr
	})

	assert.ErrorIs(t, err, retryErr)
	assert.Equal(t, 1, calls)
}

func TestRetrier_RetryOnTrue(t *testing.T) {
	retryErr := errors.New("retryable")
	r := NewRetrier(RetrierConfig{
		RetryOn:                func(err error) bool { return true },
		MaxConsecutiveAttempts: 2,
		Backoff:                time.Millisecond,
		MaxBackoff:             time.Millisecond,
	})

	calls := 0
	err := r.Execute(context.Background(), func(ctx context.Context) error {
		calls++
		return retryErr
	})

	assert.ErrorIs(t, err, retryErr)
	assert.Equal(t, 2, calls)
}

func TestRetrier_SuccessResetsFailures(t *testing.T) {
	r := NewRetrier(RetrierConfig{
		MaxConsecutiveAttempts: 2,
		Backoff:               time.Millisecond,
		MaxBackoff:            time.Millisecond,
	})

	calls := 0
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := r.Execute(ctx, func(ctx context.Context) error {
		calls++
		if calls == 1 {
			return errors.New("first fail")
		}
		if calls == 2 {
			return nil // success resets counter
		}
		if calls == 3 {
			return errors.New("another fail")
		}
		if calls == 4 {
			return nil
		}
		cancel()
		return nil
	})

	assert.ErrorIs(t, err, context.Canceled)
	assert.GreaterOrEqual(t, calls, 4)
}

func TestRetrier_BackoffCancellation(t *testing.T) {
	r := NewRetrier(RetrierConfig{
		MaxConsecutiveAttempts: 10,
		Backoff:               time.Second,
		MaxBackoff:            time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := r.Execute(ctx, func(ctx context.Context) error {
		return errors.New("fail")
	})
	elapsed := time.Since(start)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Less(t, elapsed, 500*time.Millisecond, "should exit quickly on cancel, not wait for backoff")
}
