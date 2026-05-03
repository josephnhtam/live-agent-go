package webrtc

import (
	"log/slog"

	"github.com/pion/webrtc/v3"
)

type ManagerOptions struct {
	iceServers           []webrtc.ICEServer
	pcmChannelBufferSize int
	messageBufferSize    int
	messageChannelName   string
	logger               *slog.Logger
}

func NewManagerOptions() *ManagerOptions {
	return &ManagerOptions{
		pcmChannelBufferSize: 128,
		messageBufferSize:    16,
	}
}

func (o *ManagerOptions) WithICEServers(servers []webrtc.ICEServer) *ManagerOptions {
	o.iceServers = servers
	return o
}

func (o *ManagerOptions) WithPCMChannelBufferSize(size int) *ManagerOptions {
	o.pcmChannelBufferSize = size
	return o
}

func (o *ManagerOptions) WithMessageChannelBufferSize(size int) *ManagerOptions {
	o.messageBufferSize = size
	return o
}

func (o *ManagerOptions) WithMessageChannel(name string) *ManagerOptions {
	o.messageChannelName = name
	return o
}

func (o *ManagerOptions) WithLogger(logger *slog.Logger) *ManagerOptions {
	o.logger = logger
	return o
}
