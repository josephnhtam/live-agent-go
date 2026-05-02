package voice

import (
	"errors"
	"live-agent-go/voice/internal/speech"
)

var (
	ErrInvalidVAD         = errors.New("invalid VAD")
	ErrInvalidTranscriber = errors.New("invalid Transcriber")
	ErrInvalidSynthesizer = errors.New("invalid Synthesizer")
	ErrInvalidBrain       = errors.New("invalid Brain")
	ErrNotStarted         = errors.New("agent not started")
	ErrAlreadyStarted     = errors.New("agent already started")
	ErrAlreadyStopped     = errors.New("agent already stopped")

	ErrStartingVAD         = speech.ErrStartingVAD
	ErrStartingTranscriber = speech.ErrStartingTranscriber
	ErrTranscriberStopped  = speech.ErrTranscriberStopped
	ErrVADStopped          = speech.ErrVADStopped
	ErrStartingRecognizer  = errors.New("failed to start recognizer")

	ErrSessionDone = errors.New("session done")
)
