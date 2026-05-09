package helper_test

import (
	"math"
	"testing"

	"github.com/josephnhtam/live-agent-go/voice/helper"
	"github.com/stretchr/testify/assert"
)

func TestBytesToInt16s_RoundTrip(t *testing.T) {
	original := []int16{0, 1, -1, 100, -100, math.MaxInt16, math.MinInt16}
	bytes := helper.Int16sToBytes(original)
	result := helper.BytesToInt16s(bytes)
	assert.Equal(t, original, result)
}

func TestBytesToInt16s_LittleEndian(t *testing.T) {
	data := []byte{0x00, 0x01}
	result := helper.BytesToInt16s(data)
	assert.Equal(t, []int16{256}, result)
}

func TestBytesToInt16s_OddLength(t *testing.T) {
	data := []byte{0x01, 0x00, 0xFF}
	result := helper.BytesToInt16s(data)
	assert.Equal(t, []int16{1}, result)
}

func TestBytesToInt16s_Empty(t *testing.T) {
	assert.Empty(t, helper.BytesToInt16s(nil))
}

func TestInt16sToBytes_Empty(t *testing.T) {
	assert.Empty(t, helper.Int16sToBytes(nil))
}
