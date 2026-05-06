package transcriber

import (
	"log/slog"
	"time"
)

type DeepgramTranscriberConfig struct {
	APIKey string
}

type DeepgramTranscriberOptions struct {
	model                string
	language             string
	sampleRate           int32
	bufferSize           int
	smartFormat          bool
	punctuate            bool
	endpointing          string
	utteranceEndMs       string
	profanityFilter      bool
	diarize              bool
	diarizeVersion       string
	keywords             []string
	keyterm              []string
	noDelay              bool
	fillerWords          bool
	numerals             bool
	dictation            bool
	redact               []string
	replace              []string
	search               []string
	tag                  []string
	extra                []string
	maxReconnectAttempts int
	reconnectBackoff     time.Duration
	maxReconnectBackoff  time.Duration
	logger               *slog.Logger
}

func NewDeepgramOptions() *DeepgramTranscriberOptions {
	return &DeepgramTranscriberOptions{
		model:                "nova-3",
		language:             "en-US",
		sampleRate:           48000,
		bufferSize:           128,
		punctuate:            true,
		endpointing:          "500",
		maxReconnectAttempts: 3,
		reconnectBackoff:     100 * time.Millisecond,
		maxReconnectBackoff:  5 * time.Second,
	}
}

func (o *DeepgramTranscriberOptions) WithModel(model string) *DeepgramTranscriberOptions {
	o.model = model
	return o
}

func (o *DeepgramTranscriberOptions) WithLanguage(language string) *DeepgramTranscriberOptions {
	o.language = language
	return o
}

func (o *DeepgramTranscriberOptions) WithSampleRate(rate int32) *DeepgramTranscriberOptions {
	o.sampleRate = rate
	return o
}

func (o *DeepgramTranscriberOptions) WithBufferSize(size int) *DeepgramTranscriberOptions {
	o.bufferSize = size
	return o
}

func (o *DeepgramTranscriberOptions) WithSmartFormat(enabled bool) *DeepgramTranscriberOptions {
	o.smartFormat = enabled
	return o
}

func (o *DeepgramTranscriberOptions) WithPunctuate(enabled bool) *DeepgramTranscriberOptions {
	o.punctuate = enabled
	return o
}

func (o *DeepgramTranscriberOptions) WithEndpointing(endpointing string) *DeepgramTranscriberOptions {
	o.endpointing = endpointing
	return o
}

func (o *DeepgramTranscriberOptions) WithUtteranceEndMs(ms string) *DeepgramTranscriberOptions {
	o.utteranceEndMs = ms
	return o
}

func (o *DeepgramTranscriberOptions) WithProfanityFilter(enabled bool) *DeepgramTranscriberOptions {
	o.profanityFilter = enabled
	return o
}

func (o *DeepgramTranscriberOptions) WithDiarize(enabled bool) *DeepgramTranscriberOptions {
	o.diarize = enabled
	return o
}

func (o *DeepgramTranscriberOptions) WithDiarizeVersion(version string) *DeepgramTranscriberOptions {
	o.diarizeVersion = version
	return o
}

func (o *DeepgramTranscriberOptions) WithKeywords(keywords []string) *DeepgramTranscriberOptions {
	o.keywords = keywords
	return o
}

func (o *DeepgramTranscriberOptions) WithKeyterm(keyterm []string) *DeepgramTranscriberOptions {
	o.keyterm = keyterm
	return o
}

func (o *DeepgramTranscriberOptions) WithNoDelay(enabled bool) *DeepgramTranscriberOptions {
	o.noDelay = enabled
	return o
}

func (o *DeepgramTranscriberOptions) WithFillerWords(enabled bool) *DeepgramTranscriberOptions {
	o.fillerWords = enabled
	return o
}

func (o *DeepgramTranscriberOptions) WithNumerals(enabled bool) *DeepgramTranscriberOptions {
	o.numerals = enabled
	return o
}

func (o *DeepgramTranscriberOptions) WithDictation(enabled bool) *DeepgramTranscriberOptions {
	o.dictation = enabled
	return o
}

func (o *DeepgramTranscriberOptions) WithRedact(redact []string) *DeepgramTranscriberOptions {
	o.redact = redact
	return o
}

func (o *DeepgramTranscriberOptions) WithReplace(replace []string) *DeepgramTranscriberOptions {
	o.replace = replace
	return o
}

func (o *DeepgramTranscriberOptions) WithSearch(search []string) *DeepgramTranscriberOptions {
	o.search = search
	return o
}

func (o *DeepgramTranscriberOptions) WithTag(tag []string) *DeepgramTranscriberOptions {
	o.tag = tag
	return o
}

func (o *DeepgramTranscriberOptions) WithExtra(extra []string) *DeepgramTranscriberOptions {
	o.extra = extra
	return o
}

func (o *DeepgramTranscriberOptions) WithMaxReconnectAttempts(maxAttempts int) *DeepgramTranscriberOptions {
	o.maxReconnectAttempts = maxAttempts
	return o
}

func (o *DeepgramTranscriberOptions) WithReconnectBackoff(duration time.Duration) *DeepgramTranscriberOptions {
	o.reconnectBackoff = duration
	return o
}

func (o *DeepgramTranscriberOptions) WithMaxReconnectBackoff(duration time.Duration) *DeepgramTranscriberOptions {
	o.maxReconnectBackoff = duration
	return o
}

func (o *DeepgramTranscriberOptions) WithLogger(logger *slog.Logger) *DeepgramTranscriberOptions {
	o.logger = logger
	return o
}
