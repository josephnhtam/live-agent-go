package dialog

import (
	"context"
	"live-agent-go/voice/core"
)

type Synthesizer interface {
	Synthesize(ctx context.Context, tokens <-chan core.Token) (<-chan core.AudioFrame, error)
}

type Brain interface {
	Generate(ctx context.Context, prompt string) (<-chan core.Token, error)
}
