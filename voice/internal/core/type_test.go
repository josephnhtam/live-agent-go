package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPCMFrame_Accessors(t *testing.T) {
	ctx := context.Background()
	f := &PCMFrame{
		PCMData:      []int16{1, 2, 3},
		SampleRateHz: 16000,
		NumChannels:  1,
		Ctx:          ctx,
	}

	assert.Equal(t, int32(16000), f.SampleRate())
	assert.Equal(t, int8(1), f.Channels())
	assert.Equal(t, ctx, f.Context())
}

func TestPCMFrame_SetContext(t *testing.T) {
	f := &PCMFrame{Ctx: context.Background()}
	newCtx := context.WithValue(context.Background(), "key", "value")
	f.SetContext(newCtx)
	assert.Equal(t, newCtx, f.Context())
}

func TestOpusFrame_Accessors(t *testing.T) {
	ctx := context.Background()
	f := &OpusFrame{
		OpusData:     []byte{0x01, 0x02},
		SampleRateHz: 48000,
		NumChannels:  2,
		Ctx:          ctx,
	}

	assert.Equal(t, int32(48000), f.SampleRate())
	assert.Equal(t, int8(2), f.Channels())
	assert.Equal(t, ctx, f.Context())
}

func TestOpusFrame_SetContext(t *testing.T) {
	f := &OpusFrame{Ctx: context.Background()}
	newCtx := context.WithValue(context.Background(), "key", "value")
	f.SetContext(newCtx)
	assert.Equal(t, newCtx, f.Context())
}

func TestAudioFrameInterface(t *testing.T) {
	var _ AudioFrame = (*PCMFrame)(nil)
	var _ AudioFrame = (*OpusFrame)(nil)
}
