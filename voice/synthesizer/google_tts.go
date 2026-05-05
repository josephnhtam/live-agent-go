package synthesizer

import (
	"context"
	"fmt"
	"github.com/josephnhtam/live-agent-go/voice"
	"github.com/josephnhtam/live-agent-go/voice/helper"
	"io"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
)

type GoogleSynthesizer struct {
	config  GoogleSynthesizerConfig
	options GoogleSynthesizerOptions
	logger  *slog.Logger
}

var _ voice.Synthesizer = (*GoogleSynthesizer)(nil)

func NewGoogleSynthesizer(config GoogleSynthesizerConfig, opts *GoogleSynthesizerOptions) *GoogleSynthesizer {
	options := opts
	if options == nil {
		options = NewGoogleOptions()
	}

	logger := options.logger
	if logger == nil {
		logger = helper.NoopLogger()
	}

	return &GoogleSynthesizer{
		config:  config,
		options: *options,
		logger:  logger.WithGroup("google_tts"),
	}
}

func (s *GoogleSynthesizer) Synthesize(ctx context.Context, tokens <-chan voice.Token, audioCh chan<- voice.AudioFrame) error {
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCreateClient, err)
	}

	defer client.Close()
	s.streamLoop(ctx, client, tokens, audioCh)
	return nil
}

func (s *GoogleSynthesizer) streamLoop(
	ctx context.Context,
	client *texttospeech.Client,
	tokens <-chan voice.Token,
	audioCh chan<- voice.AudioFrame,
) {
	for {
		var firstToken voice.Token
		select {
		case token, ok := <-tokens:
			if !ok {
				return
			}
			firstToken = token
		case <-ctx.Done():
			return
		}

		stream, err := s.openStream(ctx, client)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			s.logger.Error("open stream", "error", err)
			return
		}

		grp, grpCtx := errgroup.WithContext(ctx)

		grp.Go(func() error {
			return s.sendLoop(grpCtx, stream, tokens, &firstToken)
		})

		grp.Go(func() error {
			return s.recvLoop(grpCtx, stream, audioCh)
		})

		err = grp.Wait()

		if ctx.Err() != nil {
			return
		}

		if err == nil {
			return
		}

		s.logger.Warn("stream ended, reconnecting", "error", err)
	}
}

func (s *GoogleSynthesizer) openStream(
	ctx context.Context,
	client *texttospeech.Client,
) (texttospeechpb.TextToSpeech_StreamingSynthesizeClient, error) {
	stream, err := client.StreamingSynthesize(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOpenStream, err)
	}

	configReq := &texttospeechpb.StreamingSynthesizeRequest{
		StreamingRequest: &texttospeechpb.StreamingSynthesizeRequest_StreamingConfig{
			StreamingConfig: &texttospeechpb.StreamingSynthesizeConfig{
				Voice: &texttospeechpb.VoiceSelectionParams{
					LanguageCode: s.config.LanguageCode,
					Name:         s.config.VoiceName,
				},
				StreamingAudioConfig: &texttospeechpb.StreamingAudioConfig{
					AudioEncoding:   texttospeechpb.AudioEncoding_PCM,
					SampleRateHertz: s.options.sampleRate,
				},
			},
		},
	}

	if err := stream.Send(configReq); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSendConfig, err)
	}

	return stream, nil
}

func (s *GoogleSynthesizer) sendLoop(
	ctx context.Context,
	stream texttospeechpb.TextToSpeech_StreamingSynthesizeClient,
	tokens <-chan voice.Token,
	firstToken *voice.Token,
) error {
	defer stream.CloseSend()

	sb := strings.Builder{}

	if firstToken != nil {
		sb.WriteString(firstToken.Text)
		if err := s.trySendSentence(&sb, stream); err != nil {
			return err
		}
	}

	flushTimer := time.NewTimer(s.options.flushTimeout)
	defer flushTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-flushTimer.C:
			if sb.Len() > 0 {
				if err := s.sendText(sb.String(), stream); err != nil {
					return fmt.Errorf("%w: %w", ErrSendInput, err)
				}

				sb.Reset()
			}
			flushTimer.Reset(s.options.flushTimeout)

		case token, ok := <-tokens:
			if !ok {
				s.flushText(&sb, stream)
				return nil
			}

			sb.WriteString(token.Text)
			if err := s.trySendSentence(&sb, stream); err != nil {
				return err
			}

			flushTimer.Reset(s.options.flushTimeout)
		}
	}
}

func (s *GoogleSynthesizer) trySendSentence(
	sb *strings.Builder,
	stream texttospeechpb.TextToSpeech_StreamingSynthesizeClient,
) error {
	before, after, found := helper.SplitAtSentenceEnd(sb.String(), s.options.sentenceEndRunes)

	if !found {
		return nil
	}

	if err := s.sendText(before, stream); err != nil {
		return fmt.Errorf("%w: %w", ErrSendInput, err)
	}

	sb.Reset()
	sb.WriteString(after)
	return nil
}

func (s *GoogleSynthesizer) flushText(
	buf *strings.Builder,
	stream texttospeechpb.TextToSpeech_StreamingSynthesizeClient,
) {
	text := strings.TrimSpace(buf.String())
	if text == "" {
		return
	}

	if err := s.sendText(text, stream); err != nil {
		s.logger.Error("flush input", "error", err)
	}
}

func (s *GoogleSynthesizer) sendText(
	text string,
	stream texttospeechpb.TextToSpeech_StreamingSynthesizeClient,
) error {
	s.logger.Info("sending text", "text", text)
	return stream.Send(&texttospeechpb.StreamingSynthesizeRequest{
		StreamingRequest: &texttospeechpb.StreamingSynthesizeRequest_Input{
			Input: &texttospeechpb.StreamingSynthesisInput{
				InputSource: &texttospeechpb.StreamingSynthesisInput_Text{
					Text: text,
				},
			},
		},
	})
}

func (s *GoogleSynthesizer) recvLoop(
	ctx context.Context,
	stream texttospeechpb.TextToSpeech_StreamingSynthesizeClient,
	audioCh chan<- voice.AudioFrame,
) error {
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF || ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("%w: %w", ErrRecv, err)
		}

		data := resp.GetAudioContent()
		if len(data) == 0 {
			continue
		}

		samples := helper.BytesToInt16s(data)

		select {
		case audioCh <- &voice.PCMFrame{
			PCMData:      samples,
			SampleRateHz: s.options.sampleRate,
			NumChannels:  1,
		}:
		case <-ctx.Done():
			return nil
		}
	}
}
