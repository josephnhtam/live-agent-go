package helper_test

import (
	"testing"

	"github.com/josephnhtam/live-agent-go/voice/helper"
)

func TestNoopLogger_DoesNotPanic(t *testing.T) {
	logger := helper.NoopLogger()
	logger.Info("test message")
	logger.Error("test error")
	logger.Warn("test warning")
	logger.Debug("test debug")
	logger.With("key", "value").Info("with attrs")
	logger.WithGroup("group").Info("with group")
}
