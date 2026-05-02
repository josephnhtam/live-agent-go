package speech

import "errors"

var (
	ErrStartingVAD         = errors.New("failed to start VAD")
	ErrStartingTranscriber = errors.New("failed to start Transcriber")
	ErrFeedingVAD          = errors.New("failed to feed VAD")
	ErrFeedingTranscriber  = errors.New("failed to feed Transcriber")
	ErrTranscriberStopped  = errors.New("transcriber stopped")
	ErrVADStopped          = errors.New("vad stopped")
)
