package vad

import "time"

type SileroVADOptions struct {
	threshold       float32
	silenceDuration time.Duration
	eventBufferSize int
}

func NewSileroVADOptions() *SileroVADOptions {
	return &SileroVADOptions{
		threshold:       0.5,
		silenceDuration: 300 * time.Millisecond,
		eventBufferSize: 8,
	}
}

func (o *SileroVADOptions) WithThreshold(threshold float32) *SileroVADOptions {
	o.threshold = threshold
	return o
}

func (o *SileroVADOptions) WithSilenceDuration(duration time.Duration) *SileroVADOptions {
	o.silenceDuration = duration
	return o
}

func (o *SileroVADOptions) WithEventBufferSize(size int) *SileroVADOptions {
	o.eventBufferSize = size
	return o
}
