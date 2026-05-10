package transcriber

import (
	"log/slog"
	"net/http"
	"time"
)

type OpenAIRealtimeTranscriberConfig struct {
	Endpoint string
	APIKey   string
	Model    string
}

type TurnDetectionConfig struct {
	Type              string  `json:"type"`
	Threshold         float64 `json:"threshold,omitempty"`
	PrefixPaddingMs   int     `json:"prefix_padding_ms,omitempty"`
	SilenceDurationMs int     `json:"silence_duration_ms,omitempty"`
}

type OpenAIRealtimeTranscriberOptions struct {
	language             string
	prompt               string
	sampleRate           int32
	bufferSize           int
	turnDetection        *TurnDetectionConfig
	noiseReduction       string
	maxReconnectAttempts int
	reconnectBackoff     time.Duration
	maxReconnectBackoff  time.Duration
	headers              http.Header
	logger               *slog.Logger
}

func NewOpenAIRealtimeOptions() *OpenAIRealtimeTranscriberOptions {
	return &OpenAIRealtimeTranscriberOptions{
		sampleRate:           24000,
		bufferSize:           128,
		turnDetection:        &TurnDetectionConfig{Type: "server_vad"},
		maxReconnectAttempts: 3,
		reconnectBackoff:     100 * time.Millisecond,
		maxReconnectBackoff:  5 * time.Second,
	}
}

func (o *OpenAIRealtimeTranscriberOptions) WithLanguage(language string) *OpenAIRealtimeTranscriberOptions {
	o.language = language
	return o
}

func (o *OpenAIRealtimeTranscriberOptions) WithPrompt(prompt string) *OpenAIRealtimeTranscriberOptions {
	o.prompt = prompt
	return o
}

func (o *OpenAIRealtimeTranscriberOptions) WithBufferSize(size int) *OpenAIRealtimeTranscriberOptions {
	o.bufferSize = size
	return o
}

func (o *OpenAIRealtimeTranscriberOptions) WithTurnDetection(config *TurnDetectionConfig) *OpenAIRealtimeTranscriberOptions {
	c := *config
	o.turnDetection = &c
	return o
}

func (o *OpenAIRealtimeTranscriberOptions) WithNoiseReduction(noiseReduction string) *OpenAIRealtimeTranscriberOptions {
	o.noiseReduction = noiseReduction
	return o
}

func (o *OpenAIRealtimeTranscriberOptions) WithMaxReconnectAttempts(maxAttempts int) *OpenAIRealtimeTranscriberOptions {
	o.maxReconnectAttempts = maxAttempts
	return o
}

func (o *OpenAIRealtimeTranscriberOptions) WithReconnectBackoff(duration time.Duration) *OpenAIRealtimeTranscriberOptions {
	o.reconnectBackoff = duration
	return o
}

func (o *OpenAIRealtimeTranscriberOptions) WithMaxReconnectBackoff(duration time.Duration) *OpenAIRealtimeTranscriberOptions {
	o.maxReconnectBackoff = duration
	return o
}

func (o *OpenAIRealtimeTranscriberOptions) WithHeaders(headers http.Header) *OpenAIRealtimeTranscriberOptions {
	o.headers = headers
	return o
}

func (o *OpenAIRealtimeTranscriberOptions) WithLogger(logger *slog.Logger) *OpenAIRealtimeTranscriberOptions {
	o.logger = logger
	return o
}
