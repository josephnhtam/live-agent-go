package helper

import "encoding/binary"

func BytesToInt16s(data []byte) []int16 {
	samples := make([]int16, len(data)/2)

	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(data[i*2:]))
	}

	return samples
}

func Int16sToBytes(samples []int16) []byte {
	data := make([]byte, len(samples)*2)

	for i, s := range samples {
		binary.LittleEndian.PutUint16(data[i*2:], uint16(s))
	}

	return data
}
