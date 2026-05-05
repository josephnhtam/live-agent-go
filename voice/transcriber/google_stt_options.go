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
	recognizer             string
	sampleRate             int32
	bufferSize             int
	maxReconnectAttempts   int
	reconnectBackoff       time.Duration
	maxReconnectBackoff    time.Duration
	endpointingSensitivity speechpb.StreamingRecognitionFeatures_EndpointingSensitivity
	speechEndTimeout       time.Duration
	speechStartTimeout     time.Duration
	recognitionFeatures    *speechpb.RecognitionFeatures
	logger                 *slog.Logger
}

func NewGoogleOptions() *GoogleTranscriberOptions {
	return &GoogleTranscriberOptions{
		recognizer:           "_",
		sampleRate:           48000,
		bufferSize:           128,
		maxReconnectAttempts: 3,
		reconnectBackoff:     100 * time.Millisecond,
		maxReconnectBackoff:  5 * time.Second,
		recognitionFeatures: &speechpb.RecognitionFeatures{
			EnableAutomaticPunctuation: true,
		},
	}
}

func (o *GoogleTranscriberOptions) WithRecognizer(name string) *GoogleTranscriberOptions {
	o.recognizer = name
	return o
}

func (o *GoogleTranscriberOptions) WithSampleRate(rate int32) *GoogleTranscriberOptions {
	o.sampleRate = rate
	return o
}

func (o *GoogleTranscriberOptions) WithBufferSize(size int) *GoogleTranscriberOptions {
	o.bufferSize = size
	return o
}

func (o *GoogleTranscriberOptions) WithMaxReconnectAttempts(maxAttempts int) *GoogleTranscriberOptions {
	o.maxReconnectAttempts = maxAttempts
	return o
}

func (o *GoogleTranscriberOptions) WithReconnectBackoff(duration time.Duration) *GoogleTranscriberOptions {
	o.reconnectBackoff = duration
	return o
}

func (o *GoogleTranscriberOptions) WithMaxReconnectBackoff(duration time.Duration) *GoogleTranscriberOptions {
	o.maxReconnectBackoff = duration
	return o
}

func (o *GoogleTranscriberOptions) WithEndpointingSensitivity(sensitivity speechpb.StreamingRecognitionFeatures_EndpointingSensitivity) *GoogleTranscriberOptions {
	o.endpointingSensitivity = sensitivity
	return o
}

func (o *GoogleTranscriberOptions) WithSpeechEndTimeout(duration time.Duration) *GoogleTranscriberOptions {
	o.speechEndTimeout = duration
	return o
}

func (o *GoogleTranscriberOptions) WithSpeechStartTimeout(duration time.Duration) *GoogleTranscriberOptions {
	o.speechStartTimeout = duration
	return o
}

func (o *GoogleTranscriberOptions) WithRecognitionFeatures(features *speechpb.RecognitionFeatures) *GoogleTranscriberOptions {
	o.recognitionFeatures = features
	return o
}

func (o *GoogleTranscriberOptions) WithLogger(logger *slog.Logger) *GoogleTranscriberOptions {
	o.logger = logger
	return o
}
