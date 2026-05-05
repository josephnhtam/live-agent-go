package synthesizer

import "errors"

var (
	ErrCreateClient = errors.New("failed to create TTS client")
	ErrOpenStream   = errors.New("failed to open streaming synthesize")
	ErrSendConfig   = errors.New("failed to send streaming config")
	ErrSendInput    = errors.New("failed to send synthesis input")
	ErrRecv         = errors.New("failed to receive TTS audio")
)
