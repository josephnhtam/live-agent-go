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
	Generate(ctx context.Context, prompt string, tools Tools, tokens chan<- core.Token) error
}

type Tools interface {
	AddFiller(token core.Token)
	PlayAudio(wave *Wave, opts *AudioOptions) (AudioHandle, error)
}

type AudioHandle interface {
	SetVolume(v float64)
	Stop()
}
