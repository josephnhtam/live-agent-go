package helper_test

import (
	"testing"

	"github.com/josephnhtam/live-agent-go/voice/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResampleLinear_SameRate(t *testing.T) {
	data := []int16{1, 2, 3, 4, 5}
	var last int16
	result := helper.ResampleLinear(data, 16000, 16000, &last)
	assert.Same(t, &data[0], &result[0], "same rate should return original slice")
}

func TestResampleLinear_EmptyInput(t *testing.T) {
	var last int16
	result := helper.ResampleLinear(nil, 16000, 8000, &last)
	assert.Empty(t, result)
}

func TestResampleLinear_Upsample2x(t *testing.T) {
	data := []int16{0, 100}
	var last int16
	result := helper.ResampleLinear(data, 8000, 16000, &last)
	assert.Len(t, result, 4)
	assert.Equal(t, int16(100), last)
}

func TestResampleLinear_Downsample2x(t *testing.T) {
	data := []int16{0, 50, 100, 150}
	var last int16
	result := helper.ResampleLinear(data, 16000, 8000, &last)
	assert.Len(t, result, 2)
	assert.Equal(t, int16(150), last)
}

func TestResampleLinear_LastSampleContinuity(t *testing.T) {
	chunk1 := []int16{0, 100}
	chunk2 := []int16{200, 300}
	var last int16

	r1 := helper.ResampleLinear(chunk1, 8000, 16000, &last)
	assert.Equal(t, int16(100), last)

	r2 := helper.ResampleLinear(chunk2, 8000, 16000, &last)
	assert.Equal(t, int16(300), last)

	require.NotEmpty(t, r1)
	require.NotEmpty(t, r2)
}
