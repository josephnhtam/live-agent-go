package vad

import (
	"context"
	"live-agent-go/voice/core"
	"live-agent-go/voice/internal/speech"
	"time"

	"github.com/zserge/govad"
)

const (
	sampleRate      = 16000
	samplesPerFrame = govad.SamplesPerFrame
	frameDuration   = time.Duration(samplesPerFrame) * time.Second / sampleRate

	int16MaxValue = 32768.0
)

func silenceFramesFor(duration time.Duration) int {
	return int(duration / frameDuration)
}

type SileroVAD struct {
	options  SileroVADOptions
	detector *govad.VAD
	eventCh  chan speech.VADEvent

	buf    []float32
	bufLen int

	speaking      bool
	silenceFrames int
	maxSilence    int
}

var _ speech.VAD = (*SileroVAD)(nil)

func NewSileroVAD(opts ...SileroVADOption) *SileroVAD {
	options := buildSileroVADOptions(opts...)

	return &SileroVAD{
		options:    options,
		buf:        make([]float32, samplesPerFrame),
		maxSilence: silenceFramesFor(options.SilenceDuration),
	}
}

func (v *SileroVAD) Start(ctx context.Context) error {
	if v.detector != nil {
		_ = v.Stop(ctx)
	}

	detector, err := govad.New()
	if err != nil {
		return err
	}

	v.detector = detector
	v.eventCh = make(chan speech.VADEvent, v.options.EventBufferSize)
	v.bufLen = 0
	v.speaking = false
	v.silenceFrames = 0

	return nil
}

func (v *SileroVAD) Stop(_ context.Context) error {
	if v.eventCh != nil {
		close(v.eventCh)
		v.eventCh = nil
	}

	v.detector = nil
	return nil
}

func (v *SileroVAD) Feed(ctx context.Context, frame core.AudioFrame) error {
	if v.detector == nil {
		return ErrVADNotStarted
	}

	pcmFrame, ok := frame.(*core.PCMFrame)
	if !ok {
		return ErrUnsupportedFrameType
	}

	if frame.SampleRate() < sampleRate || int(frame.SampleRate())%sampleRate != 0 {
		return ErrUnsupportedSampleRate
	}

	if frame.Channels() != 1 {
		return ErrUnsupportedChannels
	}

	samples := convertAndResample(pcmFrame)

	offset := 0
	for offset < len(samples) {
		space := samplesPerFrame - v.bufLen
		remaining := len(samples) - offset

		n := space
		if remaining < n {
			n = remaining
		}

		copy(v.buf[v.bufLen:], samples[offset:offset+n])
		v.bufLen += n
		offset += n

		if v.bufLen == samplesPerFrame {
			v.processFrame(ctx)
			v.bufLen = 0
		}
	}

	return nil
}

func (v *SileroVAD) Event() <-chan speech.VADEvent {
	return v.eventCh
}

func (v *SileroVAD) processFrame(ctx context.Context) {
	prob := v.detector.Process(v.buf)

	if prob >= v.options.Threshold {
		v.silenceFrames = 0
		if !v.speaking {
			v.speaking = true
			v.emit(ctx, speech.VADEventSpeechStart)
		}
	} else if v.speaking {
		v.silenceFrames++
		if v.silenceFrames >= v.maxSilence {
			v.speaking = false
			v.silenceFrames = 0
			v.detector.Reset()
			v.emit(ctx, speech.VADEventSpeechEnd)
		}
	}
}

func (v *SileroVAD) emit(ctx context.Context, event speech.VADEvent) {
	select {
	case v.eventCh <- event:
	case <-ctx.Done():
	}
}

func convertAndResample(frame *core.PCMFrame) []float32 {
	data := frame.PCMData
	ratio := int(frame.SampleRateHz) / sampleRate

	if ratio <= 1 {
		out := make([]float32, len(data))
		for i, s := range data {
			out[i] = float32(s) / int16MaxValue
		}
		return out
	}

	out := make([]float32, len(data)/ratio)
	for i := range out {
		out[i] = float32(data[i*ratio]) / int16MaxValue
	}

	return out
}
