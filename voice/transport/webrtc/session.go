package webrtc

import (
	"context"
	"errors"
	"io"
	"live-agent-go/voice/core"
	"live-agent-go/voice/helper"
	"live-agent-go/voice/transport/types"
	"log/slog"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"gopkg.in/hraban/opus.v2"
)

type Session struct {
	pc        *webrtc.PeerConnection
	audioIn   chan core.AudioFrame
	messageIn chan string
	ctx       context.Context
	cancel    context.CancelFunc
	trackWg   sync.WaitGroup
	once      sync.Once
	logger    *slog.Logger

	outTrack           *webrtc.TrackLocalStaticSample
	encoder            *opus.Encoder
	outBuf             []int16
	outBufLen          int
	outTicker          *time.Ticker
	resampleLastSample int16
	messageChannelName string
	dc                 *webrtc.DataChannel
	messageReady       chan struct{}
}

var _ types.Session = (*Session)(nil)

func newSession(
	pc *webrtc.PeerConnection,
	messageChannelName string,
	audioBufferSize int,
	messageBufferSize int,
	logger *slog.Logger,
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

	if logger == nil {
		logger = helper.NoopLogger()
	}

	s := &Session{
		pc:                 pc,
		audioIn:            make(chan core.AudioFrame, audioBufferSize),
		messageIn:          make(chan string, messageBufferSize),
		ctx:                ctx,
		cancel:             cancel,
		logger:             logger.WithGroup("webrtc_session"),
		outTrack:           outTrack,
		encoder:            encoder,
		outBuf:             make([]int16, OpusFrameSamples),
		outTicker:          time.NewTicker(OpusFrameDuration),
		messageChannelName: messageChannelName,
		messageReady:       make(chan struct{}),
	}

	s.setupCallbacks()

	success = true
	return s, nil
}

func (s *Session) AudioIn() <-chan core.AudioFrame {
	return s.audioIn
}

func (s *Session) MessageIn() <-chan string {
	return s.messageIn
}

func (s *Session) MessageReady() <-chan struct{} {
	return s.messageReady
}

func (s *Session) SendAudio(frame core.AudioFrame) error {
	if s.ctx.Err() != nil {
		return ErrSessionClosed
	}

	data := frame.Data

	if frame.SampleRate != OpusClockRate {
		data = helper.ResampleLinear(data, frame.SampleRate, OpusClockRate, &s.resampleLastSample)
	}

	offset := 0
	for offset < len(data) {
		space := OpusFrameSamples - s.outBufLen
		remaining := len(data) - offset

		n := space
		if remaining < n {
			n = remaining
		}

		copy(s.outBuf[s.outBufLen:], data[offset:offset+n])
		s.outBufLen += n
		offset += n

		if s.outBufLen == OpusFrameSamples {
			if err := s.encodeAndSend(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Session) encodeAndSend() error {
	encoded := make([]byte, OpusMaxEncodedFrameSize)

	n, err := s.encoder.Encode(s.outBuf, encoded)
	if err != nil {
		s.outBufLen = 0
		return errors.Join(ErrOpusEncode, err)
	}

	s.outBufLen = 0

	<-s.outTicker.C

	return s.outTrack.WriteSample(media.Sample{
		Data:     encoded[:n],
		Duration: OpusFrameDuration,
	})
}

func (s *Session) SendMessage(text string) error {
	if s.ctx.Err() != nil {
		return ErrSessionClosed
	}

	if s.dc == nil {
		return ErrDataChannelNotOpen
	}

	return s.dc.SendText(text)
}

func (s *Session) Done() <-chan struct{} {
	return s.ctx.Done()
}

func (s *Session) Close() error {
	var pcErr error

	s.once.Do(func() {
		s.cancel()
		s.trackWg.Wait()

		close(s.audioIn)
		close(s.messageIn)
		s.outTicker.Stop()

		pcErr = s.pc.Close()
	})

	return pcErr
}

func (s *Session) setupCallbacks() {
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

	s.pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		if dc.Label() != s.messageChannelName {
			return
		}
		s.setupDataChannelCallbacks(dc)
	})
}

func (s *Session) setupDataChannelCallbacks(dc *webrtc.DataChannel) {
	dc.OnOpen(func() {
		s.dc = dc
		close(s.messageReady)
		s.logger.Info("data channel opened")
	})

	dc.OnClose(func() {
		s.dc = nil
		s.logger.Info("data channel closed")
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if msg.IsString {
			select {
			case s.messageIn <- string(msg.Data):
			case <-s.ctx.Done():
			default:
				s.logger.Warn("text input channel full, dropping message")
			}
		}
	})
}

func (s *Session) handleAudioTrack(track *webrtc.TrackRemote) {
	defer s.trackWg.Done()

	decoder, err := opus.NewDecoder(OpusClockRate, PCMDecoderChannels)
	if err != nil {
		s.logger.Error("failed to create Opus decoder", "error", err)
		return
	}

	pcmBuffer := make([]int16, PCMBufferSize)

	for {
		rtpPacket, _, err := track.ReadRTP()
		if err != nil {
			if err != io.EOF {
				s.logger.Error("error reading RTP", "error", err)
			}
			return
		}

		if len(rtpPacket.Payload) == 0 {
			continue
		}

		n, err := decoder.Decode(rtpPacket.Payload, pcmBuffer)
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

