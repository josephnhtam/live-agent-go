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
	SampleRate       int32
	SentenceEndRunes []rune
	FlushTimeout     time.Duration
	Logger           *slog.Logger
}

var defaultSentenceEndRunes = []rune{'.', '?', '!', '\n', '\u3002', '\uFF1F', '\uFF01'}

func NewGoogleOptions() *GoogleSynthesizerOptions {
	return &GoogleSynthesizerOptions{
		SampleRate:       24000,
		SentenceEndRunes: defaultSentenceEndRunes,
		FlushTimeout:     4 * time.Second,
	}
}

func (o *GoogleSynthesizerOptions) WithSampleRate(rate int32) *GoogleSynthesizerOptions {
	o.SampleRate = rate
	return o
}

func (o *GoogleSynthesizerOptions) WithSentenceEndRunes(runes []rune) *GoogleSynthesizerOptions {
	o.SentenceEndRunes = runes
	return o
}

func (o *GoogleSynthesizerOptions) WithFlushTimeout(d time.Duration) *GoogleSynthesizerOptions {
	o.FlushTimeout = d
	return o
}

func (o *GoogleSynthesizerOptions) WithLogger(logger *slog.Logger) *GoogleSynthesizerOptions {
	o.Logger = logger
	return o
}
