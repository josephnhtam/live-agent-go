package helper_test

import (
	"testing"
	"time"

	"github.com/josephnhtam/live-agent-go/voice/helper"
	"github.com/stretchr/testify/assert"
)

func TestTokenBucket_FirstTakeFillsToMax(t *testing.T) {
	tb := helper.NewTokenBucket(time.Millisecond, 3)
	start := time.Now()
	tb.Take()
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 5*time.Millisecond, "first take should be instant (fills to max)")
}

func TestTokenBucket_ConsumesTokensWithoutSleep(t *testing.T) {
	tb := helper.NewTokenBucket(time.Millisecond, 3)

	start := time.Now()
	tb.Take()
	tb.Take()
	tb.Take()
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 5*time.Millisecond, "3 takes from bucket of 3 should be fast")
}

func TestTokenBucket_SleepsWhenEmpty(t *testing.T) {
	tb := helper.NewTokenBucket(50*time.Millisecond, 1)

	tb.Take()

	start := time.Now()
	tb.Take()
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, 30*time.Millisecond, "should sleep when bucket is empty")
}

func TestTokenBucket_RefillsOverTime(t *testing.T) {
	tb := helper.NewTokenBucket(10*time.Millisecond, 2)

	tb.Take()
	tb.Take()

	time.Sleep(25 * time.Millisecond)

	start := time.Now()
	tb.Take()
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 15*time.Millisecond, "should have refilled after waiting")
}
