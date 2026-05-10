package synthesizer

import (
	"log/slog"
	"time"
)

type GoogleSynthesizerConfig struct {
	LanguageCode string
	VoiceName    string
}

type GoogleSynthesizerOptions struct {
	sampleRate       int32
	sentenceEndRunes []rune
	flushTimeout     time.Duration
	logger           *slog.Logger
}

func NewGoogleOptions() *GoogleSynthesizerOptions {
	return &GoogleSynthesizerOptions{
		sampleRate:       24000,
		sentenceEndRunes: defaultSentenceEndRunes,
		flushTimeout:     4 * time.Second,
	}
}

func (o *GoogleSynthesizerOptions) WithSampleRate(rate int32) *GoogleSynthesizerOptions {
	o.sampleRate = rate
	return o
}

func (o *GoogleSynthesizerOptions) WithSentenceEndRunes(runes []rune) *GoogleSynthesizerOptions {
	o.sentenceEndRunes = runes
	return o
}

func (o *GoogleSynthesizerOptions) WithFlushTimeout(d time.Duration) *GoogleSynthesizerOptions {
	o.flushTimeout = d
	return o
}

func (o *GoogleSynthesizerOptions) WithLogger(logger *slog.Logger) *GoogleSynthesizerOptions {
	o.logger = logger
	return o
}
