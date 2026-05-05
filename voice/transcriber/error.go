package transcriber

import "errors"

var (
	ErrTranscriberNotStarted = errors.New("transcriber not started")
	ErrCreateClient          = errors.New("failed to create speech client")
	ErrOpenStream            = errors.New("failed to open streaming recognize")
	ErrSendConfig            = errors.New("failed to send streaming config")
	ErrUnsupportedSampleRate = errors.New("unsupported sample rate")
	ErrUnsupportedChannels   = errors.New("unsupported channels")
	ErrCreateConnection      = errors.New("failed to create deepgram connection")
	ErrConnect               = errors.New("failed to connect to deepgram")
	ErrCreateWebSocket       = errors.New("failed to create websocket connection")
	ErrSendSessionConfig     = errors.New("failed to send session config")
	ErrUnsupportedFrameType  = errors.New("unsupported audio frame type: expected PCMFrame")
)
