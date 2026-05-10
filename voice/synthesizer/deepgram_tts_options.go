package synthesizer

import (
	"log/slog"
)

type DeepgramSynthesizerConfig struct {
	APIKey string
	Model  string
}

type DeepgramSynthesizerOptions struct {
	sampleRate       int32
	sentenceEndRunes []rune
	logger           *slog.Logger
}

func NewDeepgramSynthesizerOptions() *DeepgramSynthesizerOptions {
	return &DeepgramSynthesizerOptions{
		sampleRate:       48000,
		sentenceEndRunes: defaultSentenceEndRunes,
	}
}

func (o *DeepgramSynthesizerOptions) WithSampleRate(rate int32) *DeepgramSynthesizerOptions {
	o.sampleRate = rate
	return o
}

func (o *DeepgramSynthesizerOptions) WithSentenceEndRunes(runes []rune) *DeepgramSynthesizerOptions {
	o.sentenceEndRunes = runes
	return o
}

func (o *DeepgramSynthesizerOptions) WithLogger(logger *slog.Logger) *DeepgramSynthesizerOptions {
	o.logger = logger
	return o
}
