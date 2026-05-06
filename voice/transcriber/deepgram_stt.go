package transcriber

import (
	"context"
	"fmt"
	"github.com/josephnhtam/live-agent-go/voice"
	"log/slog"
	"sync"

	listenClient "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/listen"
	listenv1ws "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/listen/v1/websocket"
	"github.com/josephnhtam/live-agent-go/voice/helper"

	msginterfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/api/listen/v1/websocket/interfaces"
	interfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/interfaces/v1"
)

type DeepgramTranscriber struct {
	config  DeepgramTranscriberConfig
	options DeepgramTranscriberOptions
	logger  *slog.Logger

	client       *listenv1ws.WSCallback
	audioCh      chan []byte
	transcriptCh chan voice.Transcript
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

var _ voice.Transcriber = (*DeepgramTranscriber)(nil)

func NewDeepgramTranscriber(config DeepgramTranscriberConfig, opts *DeepgramTranscriberOptions) *DeepgramTranscriber {
	options := opts
	if options == nil {
		options = NewDeepgramOptions()
	}

	logger := options.logger
	if logger == nil {
		logger = helper.NoopLogger()
	}

	return &DeepgramTranscriber{
		config:  config,
		options: *options,
		logger:  logger.WithGroup("deepgram_stt"),
	}
}

func (t *DeepgramTranscriber) Start(ctx context.Context) error {
	if t.cancel != nil {
		_ = t.Stop(ctx)
	}

	streamCtx, cancel := context.WithCancel(ctx)

	tOptions := &interfaces.LiveTranscriptionOptions{
		Model:           t.options.model,
		Language:        t.options.language,
		Encoding:        "linear16",
		SampleRate:      int(t.options.sampleRate),
		Channels:        1,
		SmartFormat:     t.options.smartFormat,
		Punctuate:       t.options.punctuate,
		InterimResults:  true,
		Endpointing:     t.options.endpointing,
		UtteranceEndMs:  t.options.utteranceEndMs,
		VadEvents:       false,
		ProfanityFilter: t.options.profanityFilter,
		Diarize:         t.options.diarize,
		DiarizeVersion:  t.options.diarizeVersion,
		Keywords:        t.options.keywords,
		Keyterm:         t.options.keyterm,
		NoDelay:         t.options.noDelay,
		FillerWords:     t.options.fillerWords,
		Numerals:        t.options.numerals,
		Dictation:       t.options.dictation,
		Redact:          t.options.redact,
		Replace:         t.options.replace,
		Search:          t.options.search,
		Tag:             t.options.tag,
		Extra:           t.options.extra,
	}

	cOptions := &interfaces.ClientOptions{
		APIKey:          t.config.APIKey,
		EnableKeepAlive: true,
	}

	cb := &deepgramCallback{transcriber: t}

	client, err := listenClient.NewWSUsingCallbackWithCancel(
		streamCtx, cancel,
		t.config.APIKey, cOptions, tOptions, cb,
	)
	if err != nil {
		cancel()
		return fmt.Errorf("%w: %w", ErrCreateConnection, err)
	}

	if !client.Connect() {
		cancel()
		return ErrConnect
	}

	t.client = client
	t.audioCh = make(chan []byte, t.options.bufferSize)
	t.transcriptCh = make(chan voice.Transcript, t.options.bufferSize)
	t.cancel = cancel

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.sendLoop(streamCtx)
	}()

	return nil
}

func (t *DeepgramTranscriber) Stop(_ context.Context) error {
	if t.cancel == nil {
		return ErrTranscriberNotStarted
	}

	t.cancel()
	t.wg.Wait()

	t.client.Stop()

	t.client = nil
	t.cancel = nil
	t.audioCh = nil
	t.transcriptCh = nil

	return nil
}

func (t *DeepgramTranscriber) Feed(ctx context.Context, frame voice.AudioFrame) error {
	if t.audioCh == nil {
		return ErrTranscriberNotStarted
	}

	pcmFrame, ok := frame.(*voice.PCMFrame)
	if !ok {
		return ErrUnsupportedFrameType
	}

	if frame.SampleRate() != t.options.sampleRate {
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

func (t *DeepgramTranscriber) Transcribe() <-chan voice.Transcript {
	return t.transcriptCh
}

func (t *DeepgramTranscriber) sendLoop(ctx context.Context) {
	defer close(t.transcriptCh)

	for {
		select {
		case <-ctx.Done():
			return

		case audio, ok := <-t.audioCh:
			if !ok {
				return
			}

			err := t.client.WriteBinary(audio)
			if err != nil {
				if ctx.Err() != nil {
					return
				}

				t.logger.Error("write audio", "error", err)

				if !t.client.AttemptReconnect(ctx, int64(t.options.maxReconnectAttempts)) {
					t.logger.Error("reconnect failed")
					return
				}

				t.logger.Info("reconnected")
			}
		}
	}
}

type deepgramCallback struct {
	transcriber *DeepgramTranscriber
}

var _ msginterfaces.LiveMessageCallback = (*deepgramCallback)(nil)

func (c *deepgramCallback) Open(_ *msginterfaces.OpenResponse) error {
	c.transcriber.logger.Info("connection opened")
	return nil
}

func (c *deepgramCallback) Message(mr *msginterfaces.MessageResponse) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = nil
		}
	}()

	if len(mr.Channel.Alternatives) == 0 {
		return nil
	}

	text := mr.Channel.Alternatives[0].Transcript
	if text == "" {
		return nil
	}

	transcript := voice.Transcript{
		Text:    text,
		IsFinal: mr.IsFinal,
	}

	if transcript.IsFinal {
		c.transcriber.logger.Info("final", "text", transcript.Text)
	} else {
		c.transcriber.logger.Debug("interim", "text", transcript.Text)
	}

	select {
	case c.transcriber.transcriptCh <- transcript:
	default:
	}

	return nil
}

func (c *deepgramCallback) Metadata(_ *msginterfaces.MetadataResponse) error {
	return nil
}

func (c *deepgramCallback) SpeechStarted(_ *msginterfaces.SpeechStartedResponse) error {
	return nil
}

func (c *deepgramCallback) UtteranceEnd(_ *msginterfaces.UtteranceEndResponse) error {
	return nil
}

func (c *deepgramCallback) Close(_ *msginterfaces.CloseResponse) error {
	c.transcriber.logger.Info("connection closed")
	return nil
}

func (c *deepgramCallback) Error(er *msginterfaces.ErrorResponse) error {
	c.transcriber.logger.Error("deepgram error", "message", er.ErrMsg, "description", er.Description)
	return nil
}

func (c *deepgramCallback) UnhandledEvent(byData []byte) error {
	c.transcriber.logger.Warn("unhandled event", "data", string(byData))
	return nil
}
