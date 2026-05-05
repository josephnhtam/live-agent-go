package dialog

import (
	"github.com/josephnhtam/live-agent-go/voice/core"
)

type ResponseStream struct {
	Audio <-chan core.AudioFrame
	Token <-chan core.Token
}
