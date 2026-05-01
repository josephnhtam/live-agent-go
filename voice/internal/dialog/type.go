package dialog

import (
	"live-agent-go/voice/core"
)

type ResponseStream struct {
	Audio <-chan core.AudioFrame
	Token <-chan core.Token
}
