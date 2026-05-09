package helper_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/josephnhtam/live-agent-go/voice/helper"
	"github.com/stretchr/testify/assert"
)

func TestWaitWithCtx_WGAlreadyDone(t *testing.T) {
	wg := &sync.WaitGroup{}
	assert.NoError(t, helper.WaitWithCtx(context.Background(), wg))
}

func TestWaitWithCtx_WGCompletes(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		time.Sleep(10 * time.Millisecond)
		wg.Done()
	}()

	assert.NoError(t, helper.WaitWithCtx(context.Background(), wg))
}

func TestWaitWithCtx_ContextCanceled(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.ErrorIs(t, helper.WaitWithCtx(ctx, wg), context.Canceled)
}

func TestWaitWithCtx_ContextTimeout(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	assert.ErrorIs(t, helper.WaitWithCtx(ctx, wg), context.DeadlineExceeded)
}
