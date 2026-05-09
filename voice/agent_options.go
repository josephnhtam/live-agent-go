package voice

import (
	"log/slog"
	"time"
)

type AgentOptions struct {
	vad                   VAD
	respAudioChs          []chan<- AudioFrame
	respTokenChs          []chan<- Token
	respErrChs            []chan<- error
	promptChs             []chan<- Prompt
	minInterruptDuration  time.Duration
	interruptOnInterim    bool
	brainBufferSize       int
	mixerBufferSize       int
	synthOutBufferSize    int
	synthInBufferSize     int
	outputTokenBufferSize int
	logger                *slog.Logger
}

func NewAgentOptions() *AgentOptions {
	return &AgentOptions{
		minInterruptDuration:  300 * time.Millisecond,
		interruptOnInterim:    true,
		brainBufferSize:       32,
		mixerBufferSize:       32,
		synthOutBufferSize:    32,
		synthInBufferSize:     32,
		outputTokenBufferSize: 32,
	}
}

func (o *AgentOptions) SubscribeAudio(ch chan<- AudioFrame) *AgentOptions {
	o.respAudioChs = append(o.respAudioChs, ch)
	return o
}

func (o *AgentOptions) SubscribeToken(ch chan<- Token) *AgentOptions {
	o.respTokenChs = append(o.respTokenChs, ch)
	return o
}

func (o *AgentOptions) SubscribeErr(ch chan<- error) *AgentOptions {
	o.respErrChs = append(o.respErrChs, ch)
	return o
}

func (o *AgentOptions) SubscribePrompt(ch chan<- Prompt) *AgentOptions {
	o.promptChs = append(o.promptChs, ch)
	return o
}

func (o *AgentOptions) WithVAD(vad VAD) *AgentOptions {
	o.vad = vad
	return o
}

func (o *AgentOptions) WithMinInterruptDuration(d time.Duration) *AgentOptions {
	o.minInterruptDuration = d
	return o
}

func (o *AgentOptions) WithInterruptOnInterim(enabled bool) *AgentOptions {
	o.interruptOnInterim = enabled
	return o
}

func (o *AgentOptions) WithBrainBufferSize(size int) *AgentOptions {
	o.brainBufferSize = size
	return o
}

func (o *AgentOptions) WithMixerBufferSize(size int) *AgentOptions {
	o.mixerBufferSize = size
	return o
}

func (o *AgentOptions) WithSynthOutBufferSize(size int) *AgentOptions {
	o.synthOutBufferSize = size
	return o
}

func (o *AgentOptions) WithSynthInBufferSize(size int) *AgentOptions {
	o.synthInBufferSize = size
	return o
}

func (o *AgentOptions) WithOutputTokenBufferSize(size int) *AgentOptions {
	o.outputTokenBufferSize = size
	return o
}

func (o *AgentOptions) WithLogger(logger *slog.Logger) *AgentOptions {
	o.logger = logger
	return o
}
