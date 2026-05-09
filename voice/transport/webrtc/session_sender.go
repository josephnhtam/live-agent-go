package webrtc

import (
	"errors"
	"sync"

	"github.com/josephnhtam/live-agent-go/voice/helper"
	"github.com/josephnhtam/live-agent-go/voice/internal/core"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"gopkg.in/hraban/opus.v2"
)

type audioSender struct {
	outTrack           *webrtc.TrackLocalStaticSample
	encoder            *opus.Encoder
	outBuf             []int16
	outBufLen          int
	tokenBucket        *helper.TokenBucket
	resampleLastSample int16
	mu                 sync.Mutex
}

func newAudioSender(
	outTrack *webrtc.TrackLocalStaticSample,
	encoder *opus.Encoder,
	tokenBucket *helper.TokenBucket,
) *audioSender {
	return &audioSender{
		outTrack:    outTrack,
		encoder:     encoder,
		outBuf:      make([]int16, OpusFrameSamples),
		tokenBucket: tokenBucket,
	}
}

func (s *audioSender) SendAudio(frame core.AudioFrame, pacing bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch f := frame.(type) {
	case *core.OpusFrame:
		return s.sendOpusFrame(f, pacing)
	case *core.PCMFrame:
		return s.sendPCMFrame(f, pacing)
	default:
		return ErrUnsupportedFrameType
	}
}

func (s *audioSender) sendPCMFrame(frame *core.PCMFrame, pacing bool) error {
	data := frame.PCMData

	if frame.SampleRate() != OpusClockRate {
		data = helper.ResampleLinear(data, frame.SampleRate(), OpusClockRate, &s.resampleLastSample)
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
			if err := s.encodeAndSend(pacing); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *audioSender) sendOpusFrame(frame *core.OpusFrame, pacing bool) error {
	if pacing {
		s.paceWrite()
	}

	return s.outTrack.WriteSample(media.Sample{
		Data:     frame.OpusData,
		Duration: OpusFrameDuration,
	})
}

func (s *audioSender) encodeAndSend(pacing bool) error {
	encoded := make([]byte, OpusMaxEncodedFrameSize)

	n, err := s.encoder.Encode(s.outBuf, encoded)
	if err != nil {
		s.outBufLen = 0
		return errors.Join(ErrOpusEncode, err)
	}

	s.outBufLen = 0

	if pacing {
		s.paceWrite()
	}

	return s.outTrack.WriteSample(media.Sample{
		Data:     encoded[:n],
		Duration: OpusFrameDuration,
	})
}

func (s *audioSender) paceWrite() {
	s.tokenBucket.Take()
}
