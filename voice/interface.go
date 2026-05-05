package voice

import (
	"context"
)

type Brain interface {
	Generate(ctx context.Context, prompt string, tokens chan<- Token) error
}

type Synthesizer interface {
	Synthesize(ctx context.Context, tokens <-chan Token, audio chan<- AudioFrame) error
}

type VAD interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Feed(ctx context.Context, frame AudioFrame) error
	Event() <-chan VADEvent
}

type Transcriber interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Feed(ctx context.Context, frame AudioFrame) error
	Transcribe() <-chan Transcript
}
