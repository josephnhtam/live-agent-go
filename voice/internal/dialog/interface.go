package dialog

import (
	"context"
	"github.com/josephnhtam/live-agent-go/voice/core"
)

type Synthesizer interface {
	Synthesize(ctx context.Context, tokens <-chan core.Token, audio chan<- core.AudioFrame) error
}

type Brain interface {
	Generate(ctx context.Context, prompt string, tokens chan<- core.Token) error
}
