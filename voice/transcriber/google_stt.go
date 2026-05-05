package transcriber

import (
	"context"
	"fmt"
	"github.com/josephnhtam/live-agent-go/voice"
	"github.com/josephnhtam/live-agent-go/voice/helper"
	"io"
	"log/slog"
	"sync"

	speech "cloud.google.com/go/speech/apiv2"
	"cloud.google.com/go/speech/apiv2/speechpb"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"golang.org/x/sync/errgroup"

	intspeech "github.com/josephnhtam/live-agent-go/voice/internal/speech"
)

type GoogleTranscriber struct {
	config  GoogleTranscriberConfig
	options GoogleTranscriberOptions
	logger  *slog.Logger

	client       *speech.Client
	recognizer   string
	audioCh      chan []byte
	transcriptCh chan voice.Transcript
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

var _ intspeech.Transcriber = (*GoogleTranscriber)(nil)

func NewGoogleTranscriber(config GoogleTranscriberConfig, opts ...*GoogleTranscriberOptions) *GoogleTranscriber {
	options := NewGoogleOptions()
	if len(opts) > 0 && opts[0] != nil {
		options = opts[0]
	}

	logger := options.Logger
	if logger == nil {
		logger = helper.NoopLogger()
	}

	return &GoogleTranscriber{
		config:  config,
		options: *options,
		logger:  logger.WithGroup("google_stt"),
	}
}

func (t *GoogleTranscriber) Start(ctx context.Context) error {
	if t.cancel != nil {
		_ = t.Stop(ctx)
	}

	endpoint := fmt.Sprintf("%s-speech.googleapis.com:443", t.config.Location)

	client, err := speech.NewClient(ctx, option.WithEndpoint(endpoint))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCreateClient, err)
	}

	t.client = client
	t.recognizer = fmt.Sprintf(
		"projects/%s/locations/%s/recognizers/%s",
		t.config.Project, t.config.Location, t.options.Recognizer,
	)
	t.audioCh = make(chan []byte, t.options.BufferSize)
	t.transcriptCh = make(chan voice.Transcript, t.options.BufferSize)

	streamCtx, cancel := context.WithCancel(ctx)
	t.cancel = cancel

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.streamLoop(streamCtx)
	}()

	return nil
}

func (t *GoogleTranscriber) Stop(_ context.Context) error {
	if t.cancel == nil {
		return ErrTranscriberNotStarted
	}

	t.cancel()
	t.wg.Wait()

	t.client.Close()

	t.client = nil
	t.cancel = nil
	t.audioCh = nil
	t.transcriptCh = nil

	return nil
}

func (t *GoogleTranscriber) Feed(ctx context.Context, frame voice.AudioFrame) error {
	if t.audioCh == nil {
		return ErrTranscriberNotStarted
	}

	pcmFrame, ok := frame.(*voice.PCMFrame)
	if !ok {
		return ErrUnsupportedFrameType
	}

	if frame.SampleRate() != t.options.SampleRate {
		return ErrUnsupportedSampleRate
	}

	if frame.Channels() != 1 {
		return ErrUnsupportedChannels
	}

	select {
	case t.audioCh <- helper.Int16sToBytes(pcmFrame.PCMData):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *GoogleTranscriber) Transcribe() <-chan voice.Transcript {
	return t.transcriptCh
}

func (t *GoogleTranscriber) streamLoop(ctx context.Context) {
	defer close(t.transcriptCh)

	consecutiveFailures := 0

	for {
		stream, err := t.openStream(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			t.logger.Error("open stream", "error", err)
			consecutiveFailures++

			if t.options.MaxReconnectAttempts > 0 && consecutiveFailures > t.options.MaxReconnectAttempts {
				t.logger.Error("max reconnect attempts exceeded", "attempts", consecutiveFailures)
				return
			}

			if !backoff(ctx, consecutiveFailures, t.options.ReconnectBackoff, t.options.MaxReconnectBackoff) {
				return
			}
			continue
		}

		consecutiveFailures = 0

		grp, grpCtx := errgroup.WithContext(ctx)

		grp.Go(func() error {
			return t.sendLoop(grpCtx, stream)
		})

		grp.Go(func() error {
			return t.receiveLoop(grpCtx, stream)
		})

		err = grp.Wait()

		if ctx.Err() != nil {
			return
		}

		if !isReconnectable(err) {
			t.logger.Error("non-reconnectable error", "error", err)
			return
		}

		t.logger.Warn("stream ended, reconnecting", "error", err)

		if !backoff(ctx, 0, t.options.ReconnectBackoff, t.options.MaxReconnectBackoff) {
			return
		}
	}
}

func (t *GoogleTranscriber) openStream(ctx context.Context) (speechpb.Speech_StreamingRecognizeClient, error) {
	stream, err := t.client.StreamingRecognize(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOpenStream, err)
	}

	configReq := t.createConfigRequest()
	if err := stream.Send(configReq); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSendConfig, err)
	}

	return stream, nil
}

func (t *GoogleTranscriber) createConfigRequest() *speechpb.StreamingRecognizeRequest {
	return &speechpb.StreamingRecognizeRequest{
		Recognizer: t.recognizer,
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					DecodingConfig: &speechpb.RecognitionConfig_ExplicitDecodingConfig{
						ExplicitDecodingConfig: &speechpb.ExplicitDecodingConfig{
							Encoding:          speechpb.ExplicitDecodingConfig_LINEAR16,
							SampleRateHertz:   t.options.SampleRate,
							AudioChannelCount: 1,
						},
					},
					Model:         t.config.Model,
					LanguageCodes: t.config.Languages,
					Features:      t.options.RecognitionFeatures,
				},
				StreamingFeatures: &speechpb.StreamingRecognitionFeatures{
					InterimResults:         true,
					EndpointingSensitivity: t.options.EndpointingSensitivity,
					VoiceActivityTimeout:   t.buildVoiceActivityTimeout(),
				},
			},
		},
	}
}

func (t *GoogleTranscriber) buildVoiceActivityTimeout() *speechpb.StreamingRecognitionFeatures_VoiceActivityTimeout {
	if t.options.SpeechEndTimeout == 0 && t.options.SpeechStartTimeout == 0 {
		return nil
	}

	vat := &speechpb.StreamingRecognitionFeatures_VoiceActivityTimeout{}
	if t.options.SpeechStartTimeout > 0 {
		vat.SpeechStartTimeout = durationpb.New(t.options.SpeechStartTimeout)

	}
	if t.options.SpeechEndTimeout > 0 {
		vat.SpeechEndTimeout = durationpb.New(t.options.SpeechEndTimeout)
	}

	return vat
}

func (t *GoogleTranscriber) sendLoop(
	ctx context.Context,
	stream speechpb.Speech_StreamingRecognizeClient,
) error {
	defer stream.CloseSend()

	for {
		select {
		case <-ctx.Done():
			return nil

		case audio, ok := <-t.audioCh:
			if !ok {
				return nil
			}

			err := stream.Send(&speechpb.StreamingRecognizeRequest{
				StreamingRequest: &speechpb.StreamingRecognizeRequest_Audio{
					Audio: audio,
				},
			})

			if err != nil {
				if ctx.Err() == nil {
					t.logger.Error("send audio", "error", err)
				}

				return err
			}
		}
	}
}

func (t *GoogleTranscriber) receiveLoop(
	ctx context.Context,
	stream speechpb.Speech_StreamingRecognizeClient,
) error {
	for {
		resp, err := stream.Recv()
		if err != nil {
			return err
		}

		for _, result := range resp.GetResults() {
			alts := result.GetAlternatives()
			if len(alts) == 0 {
				continue
			}

			transcript := voice.Transcript{
				Text:    alts[0].GetTranscript(),
				IsFinal: result.GetIsFinal(),
			}

			if transcript.IsFinal {
				t.logger.Info("final", "text", transcript.Text)
			} else {
				t.logger.Debug("interim", "text", transcript.Text)
			}

			select {
			case t.transcriptCh <- transcript:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func isReconnectable(err error) bool {
	if err == nil || err == io.EOF {
		return true
	}

	errStatus, ok := status.FromError(err)
	if !ok {
		return true
	}

	switch errStatus.Code() {
	case codes.OutOfRange, codes.Unavailable, codes.Aborted, codes.DeadlineExceeded:
		return true
	case codes.Unauthenticated, codes.PermissionDenied, codes.InvalidArgument, codes.NotFound, codes.Unimplemented:
		return false
	default:
		return true
	}
}
