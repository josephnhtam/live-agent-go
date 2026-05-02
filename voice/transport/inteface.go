package transport

import "live-agent-go/voice/core"

type Session interface {
	AudioIn() <-chan core.AudioFrame
	SendAudio(frame core.AudioFrame) error
	TextIn() <-chan string
	SendText(text string) error
	Done() <-chan struct{}
	Close() error
}
