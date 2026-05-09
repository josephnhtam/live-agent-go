package dialog

import "errors"

var (
	ErrNilWave            = errors.New("wave is nil")
	ErrSampleRateMismatch = errors.New("wave sample rate does not match mixer")
)
