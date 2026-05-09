package dialog

//go:generate mockgen -source=interface.go -destination=mock_dialog/mock_dialog.go -package=mock_dialog

import (
	"context"
	"github.com/josephnhtam/live-agent-go/voice/audio"
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
	PlayAudio(wave *audio.Wave, opts *audio.Options) (audio.Handle, error)
	SetInterruptible(interruptible bool)
}
