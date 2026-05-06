package dialog

import (
	"context"
	"github.com/josephnhtam/live-agent-go/voice/internal/core"
)

type Synthesizer interface {
	SampleRate() int32
	Synthesize(ctx context.Context, tokens <-chan core.Token, audio chan<- core.AudioFrame) error
}

type Brain interface {
	Generate(ctx context.Context, prompt string, tokens chan<- core.Token) error
}
