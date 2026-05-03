package vad

import "errors"

var (
	ErrVADNotStarted        = errors.New("vad not started")
	ErrUnsupportedSampleRate = errors.New("unsupported sample rate: must be a multiple of 16000")
	ErrUnsupportedChannels   = errors.New("unsupported channels: only mono is supported")
)
