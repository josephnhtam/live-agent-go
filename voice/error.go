package voice

import (
	"errors"
)

var (
	ErrInvalidTranscriber = errors.New("invalid Transcriber")
	ErrInvalidSynthesizer = errors.New("invalid Synthesizer")
	ErrInvalidBrain       = errors.New("invalid Brain")
	ErrNotStarted         = errors.New("agent not started")
	ErrAlreadyStarted     = errors.New("agent already started")
	ErrAlreadyStopped     = errors.New("agent already stopped")
	ErrStartingRecognizer = errors.New("failed to start recognizer")
	ErrSessionDone        = errors.New("session done")
)
