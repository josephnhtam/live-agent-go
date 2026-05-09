package helper

import (
	"context"
	"sync"
)

func WaitWithCtx(ctx context.Context, wg *sync.WaitGroup) error {
	ch := make(chan struct{})

	go func() {
		wg.Wait()
		close(ch)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		return nil
	}
}
