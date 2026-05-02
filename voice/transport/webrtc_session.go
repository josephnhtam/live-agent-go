package transport

import (
	"context"
	"errors"
	"io"
	"live-agent-go/voice/core"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/kazzmir/opus-go/opus"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type WebRTCSession struct {
	pc      *webrtc.PeerConnection
	audioIn chan core.AudioFrame
	textIn  chan string
	ctx     context.Context
	cancel  context.CancelFunc
	trackWg sync.WaitGroup
	once    sync.Once
	logger  *slog.Logger

	outTrack *webrtc.TrackLocalStaticSample
	encoder  *opus.Encoder
	dc       *webrtc.DataChannel
	dcOpen   atomic.Bool
}

var _ Session = (*WebRTCSession)(nil)

func newWebRTCSession(
	pc *webrtc.PeerConnection,
	outTrack *webrtc.TrackLocalStaticSample,
	dataChannel *webrtc.DataChannel,
	audioBufferSize int,
	textBufferSize int,
	logger *slog.Logger,
) (*WebRTCSession, error) {
	ctx, cancel := context.WithCancel(context.Background())

	encoder, err := opus.NewEncoder(OpusClockRate, PCMEncoderChannels, opus.ApplicationVoIP)
	if err != nil {
		cancel()
		return nil, errors.Join(ErrCreateOpusEncoder, err)
	}

	s := &WebRTCSession{
		pc:       pc,
		audioIn:  make(chan core.AudioFrame, audioBufferSize),
		textIn:   make(chan string, textBufferSize),
		ctx:      ctx,
		cancel:   cancel,
		logger:   logger,
		outTrack: outTrack,
		encoder:  encoder,
		dc:       dataChannel,
	}

	s.setupCallbacks()

	return s, nil
}

func (s *WebRTCSession) AudioIn() <-chan core.AudioFrame {
	return s.audioIn
}

func (s *WebRTCSession) TextIn() <-chan string {
	return s.textIn
}

func (s *WebRTCSession) SendAudio(frame core.AudioFrame) error {
	if s.ctx.Err() != nil {
		return ErrSessionClosed
	}

	encoded := make([]byte, OpusMaxEncodedFrameSize)

	n, err := s.encoder.Encode(frame.Data, len(frame.Data), encoded)
	if err != nil {
		return errors.Join(ErrOpusEncode, err)
	}

	return s.outTrack.WriteSample(media.Sample{
		Data:     encoded[:n],
		Duration: OpusFrameDuration,
	})
}

func (s *WebRTCSession) SendText(text string) error {
	if s.ctx.Err() != nil {
		return ErrSessionClosed
	}

	if !s.dcOpen.Load() {
		return ErrDataChannelNotOpen
	}

	return s.dc.SendText(text)
}

func (s *WebRTCSession) Done() <-chan struct{} {
	return s.ctx.Done()
}

func (s *WebRTCSession) Close() error {
	var pcErr error

	s.once.Do(func() {
		s.cancel()
		s.trackWg.Wait()

		close(s.audioIn)
		close(s.textIn)
		_ = s.encoder.Close()
		pcErr = s.pc.Close()
	})

	return pcErr
}

func (s *WebRTCSession) setupCallbacks() {
	s.pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		if track.Codec().MimeType != webrtc.MimeTypeOpus {
			return
		}

		s.trackWg.Add(1)
		go s.handleAudioTrack(track)
	})

	s.pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		s.logger.Info("ICE connection state changed", "state", state.String())

		switch state {
		case webrtc.ICEConnectionStateDisconnected,
			webrtc.ICEConnectionStateFailed,
			webrtc.ICEConnectionStateClosed:
			go s.Close()
		}
	})

	s.dc.OnOpen(func() {
		s.dcOpen.Store(true)
		s.logger.Info("data channel opened")
	})

	s.dc.OnClose(func() {
		s.dcOpen.Store(false)
		s.logger.Info("data channel closed")
	})

	s.dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if msg.IsString {
			select {
			case s.textIn <- string(msg.Data):
			case <-s.ctx.Done():
			default:
				s.logger.Warn("text input channel full, dropping message")
			}
		}
	})
}

func (s *WebRTCSession) handleAudioTrack(track *webrtc.TrackRemote) {
	defer s.trackWg.Done()

	decoder, err := opus.NewDecoder(OpusClockRate, PCMDecoderChannels)
	if err != nil {
		s.logger.Error("failed to create Opus decoder", "error", err)
		return
	}
	defer decoder.Close()

	pcmBuffer := make([]int16, PCMBufferSize)

	for {
		rtpPacket, _, err := track.ReadRTP()
		if err != nil {
			if err != io.EOF {
				s.logger.Error("error reading RTP", "error", err)
			}
			return
		}

		n, err := decoder.Decode(rtpPacket.Payload, pcmBuffer, PCMBufferSize, false)
		if err != nil {
			s.logger.Warn("error decoding Opus", "error", err)
			continue
		}

		pcmData := make([]int16, n)
		copy(pcmData, pcmBuffer[:n])

		frame := core.AudioFrame{
			Data:       pcmData,
			SampleRate: OpusClockRate,
			Channels:   PCMDecoderChannels,
		}

		select {
		case s.audioIn <- frame:
		case <-s.ctx.Done():
			return
		default:
			s.logger.Warn("audio input channel full, dropping audio frame")
		}
	}
}
