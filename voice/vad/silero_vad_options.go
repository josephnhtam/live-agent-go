package vad

import "time"

type SileroVADOptions struct {
	Threshold       float32
	SilenceDuration time.Duration
	EventBufferSize int
}

var defaultSileroVADOptions = SileroVADOptions{
	Threshold:       0.5,
	SilenceDuration: 300 * time.Millisecond,
	EventBufferSize: 8,
}

type SileroVADOption interface {
	apply(*SileroVADOptions)
}

type SileroVADOptionFunc func(*SileroVADOptions)

func (f SileroVADOptionFunc) apply(options *SileroVADOptions) {
	f(options)
}

func buildSileroVADOptions(opts ...SileroVADOption) SileroVADOptions {
	options := defaultSileroVADOptions

	for _, opt := range opts {
		opt.apply(&options)
	}

	return options
}

func WithSileroVADThreshold(threshold float32) SileroVADOption {
	return SileroVADOptionFunc(func(options *SileroVADOptions) {
		options.Threshold = threshold
	})
}

func WithSileroVADSilenceDuration(duration time.Duration) SileroVADOption {
	return SileroVADOptionFunc(func(options *SileroVADOptions) {
		options.SilenceDuration = duration
	})
}

func WithSileroVADEventBufferSize(size int) SileroVADOption {
	return SileroVADOptionFunc(func(options *SileroVADOptions) {
		options.EventBufferSize = size
	})
}
