package dialog

import "github.com/josephnhtam/live-agent-go/voice/internal/core"

type tools struct {
	tokenOut chan<- core.Token
	mixer    *mixer
}

func newTools(tokenOut chan<- core.Token, mixer *mixer) *tools {
	return &tools{
		tokenOut: tokenOut,
		mixer:    mixer,
	}
}

var _ Tools = (*tools)(nil)

func (t *tools) AddFiller(token core.Token) {
	t.tokenOut <- token
}

func (t *tools) PlayAudio(wave *Wave, opts *AudioOptions) (AudioHandle, error) {
	return t.mixer.AddTrack(wave, opts)
}
