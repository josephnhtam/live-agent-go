package speech

//go:generate mockgen -source=interface.go -destination=mock_speech/mock_speech.go -package=mock_speech

import (
	"context"
	"github.com/josephnhtam/live-agent-go/voice/internal/core"
)

type RecognitionHandler interface {
	OnSpeechStart()
	OnSpeechEnd()
	OnInterim()
	OnSpeechRecognized(transcripts []core.Transcript)
}

type VAD interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Feed(ctx context.Context, frame core.AudioFrame) error
	Event() <-chan VADEvent
}

type Transcriber interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Feed(ctx context.Context, frame core.AudioFrame) error
	Transcribe() <-chan core.Transcript
}
