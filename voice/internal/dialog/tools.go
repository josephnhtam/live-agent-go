package dialog

import (
	"github.com/josephnhtam/live-agent-go/voice/audio"
	"github.com/josephnhtam/live-agent-go/voice/internal/core"
)

type tools struct {
	tokenOut  chan<- core.Token
	mixer     *mixer
	responder *Responder
}

func newTools(tokenOut chan<- core.Token, mixer *mixer, responder *Responder) *tools {
	return &tools{
		tokenOut:  tokenOut,
		mixer:     mixer,
		responder: responder,
	}
}

var _ Tools = (*tools)(nil)

func (t *tools) AddFiller(token core.Token) {
	t.tokenOut <- token
}

func (t *tools) PlayAudio(wave *audio.Wave, opts *audio.Options) (audio.Handle, error) {
	return t.mixer.AddTrack(wave, opts)
}

func (t *tools) SetInterruptible(interruptible bool) {
	t.responder.SetInterruptible(interruptible)
}
