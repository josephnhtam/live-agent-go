package types

import "live-agent-go/voice/core"

type Session interface {
	AudioIn() <-chan core.AudioFrame
	MessageIn() <-chan string
	MessageReady() bool
	SendAudio(frame core.AudioFrame) error
	SendMessage(text string) error
	Done() <-chan struct{}
	Close() error
}
