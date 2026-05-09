package dialog

import (
	"bytes"
	"fmt"
	"io"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/josephnhtam/live-agent-go/voice/helper"
)

type Wave struct {
	samples    []int16
	sampleRate int32
	channels   int8
}

func LoadWave(data []byte) (*Wave, error) {
	return LoadWaveFromReader(bytes.NewReader(data))
}

func LoadWaveFromReader(r io.ReadSeeker) (*Wave, error) {
	dec := wav.NewDecoder(r)
	if !dec.IsValidFile() {
		return nil, fmt.Errorf("invalid WAV file")
	}

	buf, err := dec.FullPCMBuffer()
	if err != nil {
		return nil, fmt.Errorf("decoding WAV: %w", err)
	}

	samples := toInt16Samples(buf)
	samples = downmixToMono(samples, int8(dec.NumChans))

	return &Wave{
		samples:    samples,
		sampleRate: int32(dec.SampleRate),
		channels:   1,
	}, nil
}

func LoadWaveWithSampleRate(data []byte, targetRate int32) (*Wave, error) {
	return LoadWaveFromReaderWithSampleRate(bytes.NewReader(data), targetRate)
}

func LoadWaveFromReaderWithSampleRate(r io.ReadSeeker, targetRate int32) (*Wave, error) {
	wave, err := LoadWaveFromReader(r)
	if err != nil {
		return nil, err
	}
	if targetRate > 0 && wave.sampleRate != targetRate {
		var lastSample int16
		wave.samples = helper.ResampleLinear(wave.samples, wave.sampleRate, targetRate, &lastSample)
		wave.sampleRate = targetRate
	}
	return wave, nil
}

func (w *Wave) Samples() []int16  { return w.samples }
func (w *Wave) SampleRate() int32 { return w.sampleRate }
func (w *Wave) Channels() int8    { return w.channels }

func downmixToMono(samples []int16, channels int8) []int16 {
	if channels <= 1 {
		return samples
	}

	ch := int(channels)
	n := len(samples) / ch
	out := make([]int16, n)

	for i := range out {
		var sum int32
		for c := 0; c < ch; c++ {
			sum += int32(samples[i*ch+c])
		}
		out[i] = int16(sum / int32(ch))
	}

	return out
}

func toInt16Samples(buf *audio.IntBuffer) []int16 {
	bitDepth := buf.SourceBitDepth
	samples := make([]int16, len(buf.Data))

	switch bitDepth {
	case 16:
		for i, v := range buf.Data {
			samples[i] = int16(v)
		}
	case 8:
		for i, v := range buf.Data {
			samples[i] = int16((v - 128) << 8)
		}
	case 24:
		for i, v := range buf.Data {
			samples[i] = int16(v >> 8)
		}
	case 32:
		for i, v := range buf.Data {
			samples[i] = int16(v >> 16)
		}
	default:
		for i, v := range buf.Data {
			samples[i] = int16(v)
		}
	}

	return samples
}
