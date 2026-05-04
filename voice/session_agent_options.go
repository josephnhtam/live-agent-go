package voice

type sessionAgentOptions struct {
	iceBreaking       bool
	messageSerializer MessageSerializer
}

var defaultSessionAgentOptions = sessionAgentOptions{
	messageSerializer: DefaultMessageSerializer{},
}

type SessionAgentOption interface {
	apply(*sessionAgentOptions)
}

type SessionAgentOptionFunc func(*sessionAgentOptions)

func (f SessionAgentOptionFunc) apply(options *sessionAgentOptions) {
	f(options)
}

func buildSessionAgentOptions(opts ...SessionAgentOption) *sessionAgentOptions {
	defaultOptions := defaultSessionAgentOptions
	options := &defaultOptions

	for _, opt := range opts {
		opt.apply(options)
	}

	return options
}

func WithIceBreaking() SessionAgentOption {
	return SessionAgentOptionFunc(func(options *sessionAgentOptions) {
		options.iceBreaking = true
	})
}

func WithMessageSerializer(serializer MessageSerializer) SessionAgentOption {
	return SessionAgentOptionFunc(func(options *sessionAgentOptions) {
		options.messageSerializer = serializer
	})
}
