package voice

import (
	"context"
)

type Brain interface {
	Generate(ctx context.Context, prompt string, tokens chan<- Token) error
}

type Synthesizer interface {
	SampleRate() int32
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

type Session interface {
	AudioIn() <-chan AudioFrame
	MessageIn() <-chan string
	MessageReady() <-chan struct{}
	SendAudio(frame AudioFrame, pacing bool) error
	SendMessage(text string) error
	Done() <-chan struct{}
	Close() error
}
