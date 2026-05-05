package voice

type SessionAgentOptions struct {
	iceBreaking       bool
	messageSerializer MessageSerializer
}

func NewSessionAgentOptions() *SessionAgentOptions {
	return &SessionAgentOptions{
		messageSerializer: DefaultMessageSerializer{},
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
