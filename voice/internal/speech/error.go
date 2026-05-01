package speech

import "errors"

var (
	ErrStartingVAD         = errors.New("error starting VAD")
	ErrStartingTranscriber = errors.New("error starting Transcriber")
	ErrFeedingVAD          = errors.New("error feeding VAD")
	ErrFeedingTranscriber  = errors.New("error feeding Transcriber")
)
