package voice

type SessionAgentOptions struct {
	iceBreaking       bool
	messageSerializer MessageSerializer
	audioBufferSize   int
	tokenBufferSize   int
	promptBufferSize  int
}

func NewSessionAgentOptions() *SessionAgentOptions {
	return &SessionAgentOptions{
		messageSerializer: DefaultMessageSerializer{},
		audioBufferSize:   128,
		tokenBufferSize:   32,
		promptBufferSize:  32,
	}
}

func (o *SessionAgentOptions) WithIceBreaking() *SessionAgentOptions {
	o.iceBreaking = true
	return o
}

func (o *SessionAgentOptions) WithMessageSerializer(serializer MessageSerializer) *SessionAgentOptions {
	o.messageSerializer = serializer
	return o
}

func (o *SessionAgentOptions) WithAudioBufferSize(size int) *SessionAgentOptions {
	o.audioBufferSize = size
	return o
}

func (o *SessionAgentOptions) WithTokenBufferSize(size int) *SessionAgentOptions {
	o.tokenBufferSize = size
	return o
}

func (o *SessionAgentOptions) WithPromptBufferSize(size int) *SessionAgentOptions {
	o.promptBufferSize = size
	return o
}
