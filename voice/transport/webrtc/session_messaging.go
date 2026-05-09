package webrtc

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/pion/webrtc/v3"
)

type messageHandler struct {
	channelName string
	dataChannel atomic.Pointer[webrtc.DataChannel]
	messageIn   chan string
	ready       chan struct{}
	readyOnce   sync.Once
	ctx         context.Context
	logger      *slog.Logger
}

func newMessageHandler(channelName string, bufSize int, ctx context.Context, logger *slog.Logger) *messageHandler {
	if channelName == "" {
		return nil
	}

	return &messageHandler{
		channelName: channelName,
		messageIn:   make(chan string, bufSize),
		ready:       make(chan struct{}),
		ctx:         ctx,
		logger:      logger,
	}
}

func (m *messageHandler) MessageIn() <-chan string {
	return m.messageIn
}

func (m *messageHandler) MessageReady() <-chan struct{} {
	return m.ready
}

func (m *messageHandler) SendMessage(text string) error {
	dc := m.dataChannel.Load()
	if dc == nil {
		return ErrDataChannelNotOpen
	}

	return dc.SendText(text)
}

func (m *messageHandler) setupCallbacks(dc *webrtc.DataChannel) {
	dc.OnOpen(func() {
		m.dataChannel.Store(dc)
		m.readyOnce.Do(func() { close(m.ready) })
		m.logger.Info("data channel opened")
	})

	dc.OnClose(func() {
		m.dataChannel.Store(nil)
		m.logger.Info("data channel closed")
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if msg.IsString {
			select {
			case m.messageIn <- string(msg.Data):
			case <-m.ctx.Done():
			default:
				m.logger.Warn("text input channel full, dropping message")
			}
		}
	})
}

func (m *messageHandler) Close() {
	close(m.messageIn)
}
