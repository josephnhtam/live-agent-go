package helper

func ResampleLinear(data []int16, fromRate, toRate int32, lastSample *int16) []int16 {
	if fromRate == toRate || len(data) == 0 {
		return data
	}

	ratio := float64(fromRate) / float64(toRate)
	outLen := int(float64(len(data)) / ratio)
	out := make([]int16, outLen)

	for i := range out {
		srcPos := float64(i) * ratio
		idx := int(srcPos)
		frac := srcPos - float64(idx)

		var s0, s1 int16

		if idx < len(data) {
			s0 = data[idx]
		} else {
			s0 = data[len(data)-1]
		}

		if idx+1 < len(data) {
			s1 = data[idx+1]
		} else {
			s1 = s0
		}

		if idx == 0 && frac > 0 {
			s0 = *lastSample
			s1 = data[0]
		}

		out[i] = int16(float64(s0)*(1-frac) + float64(s1)*frac)
	}

	*lastSample = data[len(data)-1]
	return out
}
