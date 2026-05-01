package voice

type agentOptions struct {
	respAudioCh chan<- AudioFrame
	respTokenCh chan<- Token
	respErrCh   chan<- error
	promptCh    chan<- string
}

type AgentOption interface {
	apply(*agentOptions)
}

type AgentOptionFunc func(*agentOptions)

func (f AgentOptionFunc) apply(options *agentOptions) {
	f(options)
}

func buildAgentOptions(opts ...AgentOption) *agentOptions {
	options := &agentOptions{}

	for _, opt := range opts {
		opt.apply(options)
	}

	return options
}

func SubscribeAudio(respAudioCh chan<- AudioFrame) AgentOption {
	return AgentOptionFunc(func(options *agentOptions) {
		options.respAudioCh = respAudioCh
	})
}

func SubscribeToken(respTokenCh chan<- Token) AgentOption {
	return AgentOptionFunc(func(options *agentOptions) {
		options.respTokenCh = respTokenCh
	})
}

func SubscribeErr(respErrCh chan<- error) AgentOption {
	return AgentOptionFunc(func(options *agentOptions) {
		options.respErrCh = respErrCh
	})
}

func SubscribePrompt(promptCh chan<- string) AgentOption {
	return AgentOptionFunc(func(options *agentOptions) {
		options.promptCh = promptCh
	})
}
