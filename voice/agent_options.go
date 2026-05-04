package voice

import "time"

type agentOptions struct {
	vad                  VAD
	respAudioChs         []chan<- AudioFrame
	respTokenChs         []chan<- Token
	respErrChs           []chan<- error
	promptChs            []chan<- Prompt
	minInterruptDuration time.Duration
	interruptOnInterim bool

	brainBufferSize       int
	synthesizerBufferSize int
	synTokenBufferSize    int
	outputTokenBufferSize int
}

var defaultAgentOptions = agentOptions{
	minInterruptDuration: 300 * time.Millisecond,
	interruptOnInterim:   true,
}

type AgentOption interface {
	apply(*agentOptions)
}

type AgentOptionFunc func(*agentOptions)

func (f AgentOptionFunc) apply(options *agentOptions) {
	f(options)
}

func buildAgentOptions(opts ...AgentOption) *agentOptions {
	defaultOptions := defaultAgentOptions
	options := &defaultOptions

	for _, opt := range opts {
		opt.apply(options)
	}

	return options
}

func SubscribeAudio(ch chan<- AudioFrame) AgentOption {
	return AgentOptionFunc(func(options *agentOptions) {
		options.respAudioChs = append(options.respAudioChs, ch)
	})
}

func SubscribeToken(ch chan<- Token) AgentOption {
	return AgentOptionFunc(func(options *agentOptions) {
		options.respTokenChs = append(options.respTokenChs, ch)
	})
}

func SubscribeErr(ch chan<- error) AgentOption {
	return AgentOptionFunc(func(options *agentOptions) {
		options.respErrChs = append(options.respErrChs, ch)
	})
}

func SubscribePrompt(ch chan<- Prompt) AgentOption {
	return AgentOptionFunc(func(options *agentOptions) {
		options.promptChs = append(options.promptChs, ch)
	})
}

func WithVAD(vad VAD) AgentOption {
	return AgentOptionFunc(func(options *agentOptions) {
		options.vad = vad
	})
}

func WithMinInterruptDuration(d time.Duration) AgentOption {
	return AgentOptionFunc(func(options *agentOptions) {
		options.minInterruptDuration = d
	})
}

func WithInterruptOnInterim(enabled bool) AgentOption {
	return AgentOptionFunc(func(options *agentOptions) {
		options.interruptOnInterim = enabled
	})
}

func WithBrainBufferSize(size int) AgentOption {
	return AgentOptionFunc(func(options *agentOptions) {
		options.brainBufferSize = size
	})
}

func WithSynthesizerBufferSize(size int) AgentOption {
	return AgentOptionFunc(func(options *agentOptions) {
		options.synthesizerBufferSize = size
	})
}

func WithSynTokenBufferSize(size int) AgentOption {
	return AgentOptionFunc(func(options *agentOptions) {
		options.synTokenBufferSize = size
	})
}

func WithOutputTokenBufferSize(size int) AgentOption {
	return AgentOptionFunc(func(options *agentOptions) {
		options.outputTokenBufferSize = size
	})
}
