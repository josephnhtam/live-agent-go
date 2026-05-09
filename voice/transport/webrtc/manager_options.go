package webrtc

import (
	"log/slog"
	"time"

	"github.com/pion/webrtc/v3"
)

type AudioInEncoding int

const (
	AudioInEncodingPCM  AudioInEncoding = iota
	AudioInEncodingOpus
)

type ManagerOptions struct {
	iceServers         []webrtc.ICEServer
	audioBufferSize    int
	messageBufferSize  int
	messageChannelName string
	audioInEncoding    AudioInEncoding
	pacingBurst        int
	connectionTimeout  time.Duration
	logger             *slog.Logger
}

func NewManagerOptions() *ManagerOptions {
	return &ManagerOptions{
		audioBufferSize:   128,
		messageBufferSize: 16,
		pacingBurst:       3,
		connectionTimeout: 30 * time.Second,
	}
}

func (o *ManagerOptions) WithICEServers(servers []webrtc.ICEServer) *ManagerOptions {
	o.iceServers = servers
	return o
}

func (o *ManagerOptions) WithAudioBufferSize(size int) *ManagerOptions {
	o.audioBufferSize = size
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

func (o *ManagerOptions) WithAudioInEncoding(encoding AudioInEncoding) *ManagerOptions {
	o.audioInEncoding = encoding
	return o
}

func (o *ManagerOptions) WithPacingBurst(n int) *ManagerOptions {
	o.pacingBurst = n
	return o
}

func (o *ManagerOptions) WithConnectionTimeout(d time.Duration) *ManagerOptions {
	o.connectionTimeout = d
	return o
}

func (o *ManagerOptions) WithLogger(logger *slog.Logger) *ManagerOptions {
	o.logger = logger
	return o
}
