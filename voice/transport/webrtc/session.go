package webrtc

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/josephnhtam/live-agent-go/voice"
	"github.com/josephnhtam/live-agent-go/voice/helper"
	"github.com/josephnhtam/live-agent-go/voice/internal/core"

	"github.com/pion/webrtc/v3"
	"gopkg.in/hraban/opus.v2"
)

type Session struct {
	pc        *webrtc.PeerConnection
	sender    *audioSender
	receiver  *audioReceiver
	messaging *messageHandler
	ctx       context.Context
	cancel    context.CancelFunc
	once      sync.Once
	connected chan struct{}
	connOnce  sync.Once
	logger    *slog.Logger
}

var _ voice.Session = (*Session)(nil)

func newSession(
	pc *webrtc.PeerConnection,
	options *ManagerOptions,
) (*Session, error) {
	success := false
	ctx, cancel := context.WithCancel(context.Background())

	defer func() {
		if !success {
			cancel()
		}
	}()

	outTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   OpusClockRate,
			Channels:    OpusChannels,
			SDPFmtpLine: OpusSDPFmtpLine,
		},
		"audio",
		"stream",
	)
	if err != nil {
		return nil, errors.Join(ErrAddTrack, err)
	}

	if _, err := pc.AddTrack(outTrack); err != nil {
		return nil, errors.Join(ErrAddTrack, err)
	}

	encoder, err := opus.NewEncoder(OpusClockRate, PCMEncoderChannels, opus.AppVoIP)
	if err != nil {
		return nil, errors.Join(ErrCreateOpusEncoder, err)
	}

	if options.opusBitrate > 0 {
		if err := encoder.SetBitrate(options.opusBitrate); err != nil {
			return nil, errors.Join(ErrCreateOpusEncoder, err)
		}
	}
	if options.opusComplexity > 0 {
		if err := encoder.SetComplexity(options.opusComplexity); err != nil {
			return nil, errors.Join(ErrCreateOpusEncoder, err)
		}
	}
	if options.opusMaxBandwidth > 0 {
		if err := encoder.SetMaxBandwidth(options.opusMaxBandwidth); err != nil {
			return nil, errors.Join(ErrCreateOpusEncoder, err)
		}
	}

	logger := options.logger
	if logger == nil {
		logger = helper.NoopLogger()
	}
	logger = logger.WithGroup("webrtc_session")

	s := &Session{
		pc:        pc,
		sender:    newAudioSender(outTrack, encoder, helper.NewTokenBucket(OpusFrameDuration, options.pacingBurst)),
		receiver:  newAudioReceiver(options.audioBufferSize, options.audioInEncoding, ctx, logger),
		messaging: newMessageHandler(options.messageChannelName, options.messageBufferSize, ctx, logger),
		ctx:       ctx,
		cancel:    cancel,
		connected: make(chan struct{}),
		logger:    logger,
	}

	s.setupCallbacks()

	if options.connectionTimeout > 0 {
		go s.connectionTimeoutLoop(options.connectionTimeout)
	}

	success = true
	return s, nil
}

func (s *Session) AudioIn() <-chan core.AudioFrame {
	return s.receiver.AudioIn()
}

func (s *Session) MessageIn() <-chan string {
	if s.messaging == nil {
		return nil
	}
	return s.messaging.MessageIn()
}

func (s *Session) MessageReady() <-chan struct{} {
	if s.messaging == nil {
		return nil
	}
	return s.messaging.MessageReady()
}

func (s *Session) Connected() <-chan struct{} {
	return s.connected
}

func (s *Session) SendAudio(frame core.AudioFrame, pacing bool) error {
	if s.ctx.Err() != nil {
		return ErrSessionClosed
	}

	return s.sender.SendAudio(frame, pacing)
}

func (s *Session) SendMessage(text string) error {
	if s.ctx.Err() != nil {
		return ErrSessionClosed
	}

	if s.messaging == nil {
		return ErrDataChannelNotOpen
	}

	return s.messaging.SendMessage(text)
}

func (s *Session) Done() <-chan struct{} {
	return s.ctx.Done()
}

func (s *Session) Close(ctx context.Context) error {
	var closeErr error

	s.once.Do(func() {
		s.cancel()

		pcErr := s.pc.Close()
		wgErr := s.receiver.Wait(ctx)

		s.receiver.Close()
		if s.messaging != nil {
			s.messaging.Close()
		}

		closeErr = errors.Join(pcErr, wgErr)
	})

	return closeErr
}

func (s *Session) closeAsync() {
	go s.Close(context.Background())
}

func (s *Session) connectionTimeoutLoop(timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-s.connected:
	case <-s.ctx.Done():
	case <-timer.C:
		s.logger.Warn("connection timeout", "timeout", timeout)
		s.closeAsync()
	}
}

func (s *Session) setupCallbacks() {
	s.pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		s.receiver.OnTrack(track)
	})

	s.pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		s.logger.Info("ICE connection state changed", "state", state.String())

		switch state {
		case webrtc.ICEConnectionStateConnected:
			s.connOnce.Do(func() { close(s.connected) })
		case webrtc.ICEConnectionStateDisconnected,
			webrtc.ICEConnectionStateFailed,
			webrtc.ICEConnectionStateClosed:
			s.closeAsync()
		}
	})

	if s.messaging != nil {
		s.pc.OnDataChannel(func(dc *webrtc.DataChannel) {
			if dc.Label() != s.messaging.channelName {
				return
			}
			s.messaging.setupCallbacks(dc)
		})
	}
}
