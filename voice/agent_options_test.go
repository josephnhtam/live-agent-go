package voice

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAgentOptions_Defaults(t *testing.T) {
	opts := NewAgentOptions()

	assert.Equal(t, 300*time.Millisecond, opts.minInterruptDuration)
	assert.True(t, opts.interruptOnInterim)
	assert.Equal(t, 32, opts.brainBufferSize)
	assert.Equal(t, 32, opts.mixerBufferSize)
	assert.Equal(t, 32, opts.synthOutBufferSize)
	assert.Equal(t, 32, opts.synthInBufferSize)
	assert.Equal(t, 32, opts.outputTokenBufferSize)
	assert.Nil(t, opts.logger)
	assert.Nil(t, opts.vad)
}

func TestAgentOptions_SubscribeAudio(t *testing.T) {
	opts := NewAgentOptions()
	ch1 := make(chan<- AudioFrame)
	ch2 := make(chan<- AudioFrame)

	result := opts.SubscribeAudio(ch1).SubscribeAudio(ch2)
	assert.Same(t, opts, result)
	assert.Len(t, opts.respAudioChs, 2)
}

func TestAgentOptions_SubscribeToken(t *testing.T) {
	opts := NewAgentOptions()
	ch := make(chan<- Token)
	opts.SubscribeToken(ch)
	assert.Len(t, opts.respTokenChs, 1)
}

func TestAgentOptions_SubscribeErr(t *testing.T) {
	opts := NewAgentOptions()
	ch := make(chan<- error)
	opts.SubscribeErr(ch)
	assert.Len(t, opts.respErrChs, 1)
}

func TestAgentOptions_SubscribePrompt(t *testing.T) {
	opts := NewAgentOptions()
	ch := make(chan<- Prompt)
	opts.SubscribePrompt(ch)
	assert.Len(t, opts.promptChs, 1)
}

func TestAgentOptions_WithLogger(t *testing.T) {
	opts := NewAgentOptions()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	result := opts.WithLogger(logger)
	assert.Same(t, opts, result)
	assert.Equal(t, logger, opts.logger)
}

func TestAgentOptions_WithMinInterruptDuration(t *testing.T) {
	opts := NewAgentOptions()
	opts.WithMinInterruptDuration(500 * time.Millisecond)
	assert.Equal(t, 500*time.Millisecond, opts.minInterruptDuration)
}

func TestAgentOptions_WithInterruptOnInterim(t *testing.T) {
	opts := NewAgentOptions()
	opts.WithInterruptOnInterim(false)
	assert.False(t, opts.interruptOnInterim)
}

func TestAgentOptions_BufferSizeSetters(t *testing.T) {
	opts := NewAgentOptions()
	opts.WithBrainBufferSize(64)
	opts.WithMixerBufferSize(64)
	opts.WithSynthOutBufferSize(64)
	opts.WithSynthInBufferSize(64)
	opts.WithOutputTokenBufferSize(64)

	assert.Equal(t, 64, opts.brainBufferSize)
	assert.Equal(t, 64, opts.mixerBufferSize)
	assert.Equal(t, 64, opts.synthOutBufferSize)
	assert.Equal(t, 64, opts.synthInBufferSize)
	assert.Equal(t, 64, opts.outputTokenBufferSize)
}
