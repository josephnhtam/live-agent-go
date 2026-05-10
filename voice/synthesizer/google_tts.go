package synthesizer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/josephnhtam/live-agent-go/voice"
	"github.com/josephnhtam/live-agent-go/voice/helper"

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

func (s *GoogleSynthesizer) SampleRate() int32 {
	return s.options.sampleRate
}

func (s *GoogleSynthesizer) Close(_ context.Context) error {
	return nil
}

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
	var pendingText string

	for {
		firstToken, ok := s.nextToken(ctx, tokens, &pendingText)
		if !ok {
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

		var remaining string
		grp, grpCtx := errgroup.WithContext(ctx)

		grp.Go(func() error {
			var err error
			remaining, err = s.sendLoop(grpCtx, stream, tokens, &firstToken)
			return err
		})

		grp.Go(func() error {
			return s.recvLoop(grpCtx, ctx, stream, audioCh)
		})

		err = grp.Wait()
		pendingText = remaining

		if ctx.Err() != nil {
			return
		}

		if err == nil {
			return
		}

		s.logger.Warn("stream ended, reconnecting", "error", err)
	}
}

func (s *GoogleSynthesizer) nextToken(
	ctx context.Context,
	tokens <-chan voice.Token,
	pendingText *string,
) (voice.Token, bool) {
	if *pendingText != "" {
		token := voice.Token{Text: *pendingText}
		*pendingText = ""
		return token, true
	}

	select {
	case token, ok := <-tokens:
		return token, ok
	case <-ctx.Done():
		return voice.Token{}, false
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
) (remaining string, err error) {
	defer stream.CloseSend()

	sb := strings.Builder{}
	defer func() {
		remaining = sb.String()
	}()

	if firstToken != nil {
		sb.WriteString(firstToken.Text)
		if err := s.trySendSentence(&sb, stream); err != nil {
			return "", err
		}
	}

	flushTimer := time.NewTimer(s.options.flushTimeout)
	defer flushTimer.Stop()

	for {
		if ctx.Err() != nil {
			return "", nil
		}

		select {
		case <-ctx.Done():
			return "", nil

		case <-flushTimer.C:
			s.flushText(&sb, stream)
			flushTimer.Reset(s.options.flushTimeout)

		case token, ok := <-tokens:
			if !ok {
				s.flushText(&sb, stream)
				return "", nil
			}

			sb.WriteString(token.Text)
			if err := s.trySendSentence(&sb, stream); err != nil {
				return "", err
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
	sb *strings.Builder,
	stream texttospeechpb.TextToSpeech_StreamingSynthesizeClient,
) {
	if sb.Len() == 0 {
		return
	}

	if err := s.sendText(sb.String(), stream); err != nil {
		s.logger.Error("flush input", "error", err)
	}

	sb.Reset()
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
	frameCtx context.Context,
	stream texttospeechpb.TextToSpeech_StreamingSynthesizeClient,
	audioCh chan<- voice.AudioFrame,
) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		resp, err := stream.Recv()

		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
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
			Ctx:          frameCtx,
		}:
		case <-ctx.Done():
			return nil
		}
	}
}
