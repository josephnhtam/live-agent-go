package audio

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildWAV(sampleRate uint32, numChannels, bitsPerSample uint16, samples []int16) []byte {
	dataSize := uint32(len(samples)) * uint32(bitsPerSample/8)
	blockAlign := numChannels * (bitsPerSample / 8)
	byteRate := sampleRate * uint32(blockAlign)
	fileSize := 36 + dataSize

	buf := &bytes.Buffer{}

	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, fileSize)
	buf.WriteString("WAVE")

	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16))
	binary.Write(buf, binary.LittleEndian, uint16(1))
	binary.Write(buf, binary.LittleEndian, numChannels)
	binary.Write(buf, binary.LittleEndian, sampleRate)
	binary.Write(buf, binary.LittleEndian, byteRate)
	binary.Write(buf, binary.LittleEndian, blockAlign)
	binary.Write(buf, binary.LittleEndian, bitsPerSample)

	buf.WriteString("data")
	binary.Write(buf, binary.LittleEndian, dataSize)
	for _, s := range samples {
		binary.Write(buf, binary.LittleEndian, s)
	}

	return buf.Bytes()
}

func TestLoadWave_Mono16Bit(t *testing.T) {
	samples := []int16{0, 100, -100, 32767, -32768}
	data := buildWAV(16000, 1, 16, samples)

	w, err := LoadWave(data)
	require.NoError(t, err)
	assert.Equal(t, int32(16000), w.SampleRate())
	assert.Equal(t, int8(1), w.Channels())
	assert.Equal(t, samples, w.Samples())
}

func TestLoadWave_StereoDownmixToMono(t *testing.T) {
	stereoSamples := []int16{100, 200, 0, 0}
	data := buildWAV(16000, 2, 16, stereoSamples)

	w, err := LoadWave(data)
	require.NoError(t, err)
	assert.Equal(t, int8(1), w.Channels())
	require.Len(t, w.Samples(), 2)
	assert.Equal(t, int16(150), w.Samples()[0])
}

func TestLoadWave_InvalidData(t *testing.T) {
	_, err := LoadWave([]byte("not a wav file"))
	assert.Error(t, err)
}

func TestLoadWave_EmptyData(t *testing.T) {
	_, err := LoadWave(nil)
	assert.Error(t, err)
}

func TestLoadWaveWithSampleRate_Resample(t *testing.T) {
	samples := []int16{0, 100, 200, 300}
	data := buildWAV(16000, 1, 16, samples)

	w, err := LoadWaveWithSampleRate(data, 8000)
	require.NoError(t, err)
	assert.Equal(t, int32(8000), w.SampleRate())
	assert.Len(t, w.Samples(), 2)
}

func TestLoadWaveWithSampleRate_SameRate(t *testing.T) {
	samples := []int16{10, 20, 30}
	data := buildWAV(16000, 1, 16, samples)

	w, err := LoadWaveWithSampleRate(data, 16000)
	require.NoError(t, err)
	assert.Len(t, w.Samples(), len(samples))
}

func TestLoadWaveWithSampleRate_ZeroTargetRate(t *testing.T) {
	samples := []int16{10, 20, 30}
	data := buildWAV(16000, 1, 16, samples)

	w, err := LoadWaveWithSampleRate(data, 0)
	require.NoError(t, err)
	assert.Len(t, w.Samples(), len(samples))
}

func TestLoadWaveFromReader(t *testing.T) {
	samples := []int16{42, -42}
	data := buildWAV(44100, 1, 16, samples)

	w, err := LoadWaveFromReader(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, int32(44100), w.SampleRate())
}
