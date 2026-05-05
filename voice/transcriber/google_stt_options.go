package transcriber

import (
	"log/slog"
	"time"

	speechpb "cloud.google.com/go/speech/apiv2/speechpb"
)

type GoogleTranscriberConfig struct {
	Project   string
	Location  string
	Languages []string
	Model     string
}

type GoogleTranscriberOptions struct {
	Recognizer             string
	SampleRate             int32
	BufferSize             int
	MaxReconnectAttempts   int
	ReconnectBackoff       time.Duration
	MaxReconnectBackoff    time.Duration
	EndpointingSensitivity speechpb.StreamingRecognitionFeatures_EndpointingSensitivity
	SpeechEndTimeout       time.Duration
	SpeechStartTimeout     time.Duration
	RecognitionFeatures    *speechpb.RecognitionFeatures
	Logger                 *slog.Logger
}

func NewGoogleOptions() *GoogleTranscriberOptions {
	return &GoogleTranscriberOptions{
		Recognizer:           "_",
		SampleRate:           48000,
		BufferSize:           128,
		MaxReconnectAttempts: 3,
		ReconnectBackoff:     100 * time.Millisecond,
		MaxReconnectBackoff:  5 * time.Second,
		RecognitionFeatures: &speechpb.RecognitionFeatures{
			EnableAutomaticPunctuation: true,
		},
	}
}

func (o *GoogleTranscriberOptions) WithRecognizer(name string) *GoogleTranscriberOptions {
	o.Recognizer = name
	return o
}

func (o *GoogleTranscriberOptions) WithSampleRate(rate int32) *GoogleTranscriberOptions {
	o.SampleRate = rate
	return o
}

func (o *GoogleTranscriberOptions) WithBufferSize(size int) *GoogleTranscriberOptions {
	o.BufferSize = size
	return o
}

func (o *GoogleTranscriberOptions) WithMaxReconnectAttempts(maxAttempts int) *GoogleTranscriberOptions {
	o.MaxReconnectAttempts = maxAttempts
	return o
}

func (o *GoogleTranscriberOptions) WithReconnectBackoff(duration time.Duration) *GoogleTranscriberOptions {
	o.ReconnectBackoff = duration
	return o
}

func (o *GoogleTranscriberOptions) WithMaxReconnectBackoff(duration time.Duration) *GoogleTranscriberOptions {
	o.MaxReconnectBackoff = duration
	return o
}

func (o *GoogleTranscriberOptions) WithEndpointingSensitivity(sensitivity speechpb.StreamingRecognitionFeatures_EndpointingSensitivity) *GoogleTranscriberOptions {
	o.EndpointingSensitivity = sensitivity
	return o
}

func (o *GoogleTranscriberOptions) WithSpeechEndTimeout(duration time.Duration) *GoogleTranscriberOptions {
	o.SpeechEndTimeout = duration
	return o
}

func (o *GoogleTranscriberOptions) WithSpeechStartTimeout(duration time.Duration) *GoogleTranscriberOptions {
	o.SpeechStartTimeout = duration
	return o
}

func (o *GoogleTranscriberOptions) WithRecognitionFeatures(features *speechpb.RecognitionFeatures) *GoogleTranscriberOptions {
	o.RecognitionFeatures = features
	return o
}

func (o *GoogleTranscriberOptions) WithLogger(logger *slog.Logger) *GoogleTranscriberOptions {
	o.Logger = logger
	return o
}
