package transcriber

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/josephnhtam/live-agent-go/voice"
	"github.com/josephnhtam/live-agent-go/voice/helper"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net/http"
	"sync"
)

const oaiSampleRate int32 = 24000

type OpenAIRealtimeTranscriber struct {
	config  OpenAIRealtimeTranscriberConfig
	options OpenAIRealtimeTranscriberOptions
	logger  *slog.Logger

	audioCh            chan []byte
	transcriptCh       chan voice.Transcript
	cancel             context.CancelFunc
	wg                 sync.WaitGroup
	resampleLastSample int16
}

var _ voice.Transcriber = (*OpenAIRealtimeTranscriber)(nil)

func NewOpenAIRealtimeTranscriber(config OpenAIRealtimeTranscriberConfig, opts *OpenAIRealtimeTranscriberOptions) *OpenAIRealtimeTranscriber {
	options := opts
	if options == nil {
		options = NewOpenAIRealtimeOptions()
	}

	logger := options.logger
	if logger == nil {
		logger = helper.NoopLogger()
	}

	return &OpenAIRealtimeTranscriber{
		config:  config,
		options: *options,
		logger:  logger.WithGroup("openai_realtime_stt"),
	}
}

func (t *OpenAIRealtimeTranscriber) Start(ctx context.Context) error {
	if t.cancel != nil {
		_ = t.Stop(ctx)
	}

	streamCtx, cancel := context.WithCancel(ctx)

	t.audioCh = make(chan []byte, t.options.bufferSize)
	t.transcriptCh = make(chan voice.Transcript, t.options.bufferSize)
	t.cancel = cancel
	t.resampleLastSample = 0

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.streamLoop(streamCtx)
	}()

	return nil
}

func (t *OpenAIRealtimeTranscriber) Stop(_ context.Context) error {
	if t.cancel == nil {
		return ErrTranscriberNotStarted
	}

	t.cancel()
	t.wg.Wait()

	t.cancel = nil
	t.audioCh = nil
	t.transcriptCh = nil

	return nil
}

func (t *OpenAIRealtimeTranscriber) Feed(ctx context.Context, frame voice.AudioFrame) error {
	if t.audioCh == nil {
		return ErrTranscriberNotStarted
	}

	pcmFrame, ok := frame.(*voice.PCMFrame)
	if !ok {
		return ErrUnsupportedFrameType
	}

	if frame.Channels() != 1 {
		return ErrUnsupportedChannels
	}

	data := pcmFrame.PCMData
	if frame.SampleRate() != oaiSampleRate {
		data = helper.ResampleLinear(data, frame.SampleRate(), oaiSampleRate, &t.resampleLastSample)
	}

	select {
	case t.audioCh <- helper.Int16sToBytes(data):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *OpenAIRealtimeTranscriber) Transcribe() <-chan voice.Transcript {
	return t.transcriptCh
}

func (t *OpenAIRealtimeTranscriber) streamLoop(ctx context.Context) {
	defer close(t.transcriptCh)

	r := helper.NewRetrier(helper.RetrierConfig{
		RetryOn:                t.isReconnectable,
		MaxConsecutiveAttempts: t.options.maxReconnectAttempts,
		Backoff:                t.options.reconnectBackoff,
		MaxBackoff:             t.options.maxReconnectBackoff,
	})

	if err := r.Execute(ctx, func(ctx context.Context) error {
		conn, err := t.openConnection(ctx)
		if err != nil {
			t.logger.Error("open connection", "error", err)
			return err
		}

		grp, grpCtx := errgroup.WithContext(ctx)

		grp.Go(func() error {
			return t.sendLoop(grpCtx, conn)
		})

		grp.Go(func() error {
			return t.receiveLoop(grpCtx, conn)
		})

		return grp.Wait()
	}); err != nil && ctx.Err() == nil {
		t.logger.Error("stream loop exited", "error", err)
	}
}

func (t *OpenAIRealtimeTranscriber) openConnection(ctx context.Context) (*websocket.Conn, error) {
	header := http.Header{}
	header.Set("api-key", t.config.APIKey)

	for k, vs := range t.options.headers {
		for _, v := range vs {
			header.Add(k, v)
		}
	}

	dialer := websocket.Dialer{}
	conn, resp, err := dialer.DialContext(ctx, t.config.Endpoint, header)
	if err != nil {
		if resp != nil {
			t.logger.Error("websocket handshake rejected", "status", resp.StatusCode)
		}
		return nil, fmt.Errorf("%w: %w", ErrCreateWebSocket, err)
	}

	if err := t.sendSessionConfig(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("%w: %w", ErrSendSessionConfig, err)
	}

	t.logger.Info("connection opened")
	return conn, nil
}

func (t *OpenAIRealtimeTranscriber) sendSessionConfig(conn *websocket.Conn) error {
	td := &sessionTurnDetection{
		Type:           "server_vad",
		CreateResponse: true,
	}
	if t.options.turnDetection != nil {
		td.Type = t.options.turnDetection.Type
		td.Threshold = t.options.turnDetection.Threshold
		td.PrefixPaddingMs = t.options.turnDetection.PrefixPaddingMs
		td.SilenceDurationMs = t.options.turnDetection.SilenceDurationMs
	}

	var nr *sessionNoiseReduction
	if t.options.noiseReduction != "" {
		nr = &sessionNoiseReduction{Type: t.options.noiseReduction}
	}

	msg := sessionUpdateMessage{
		Type: "session.update",
		Session: sessionUpdateSession{
			Type: "realtime",
			Audio: sessionUpdateAudio{
				Input: sessionUpdateInput{
					Transcription: sessionTranscription{
						Model:    t.config.Model,
						Language: t.options.language,
						Prompt:   t.options.prompt,
					},
					TurnDetection:  td,
					NoiseReduction: nr,
				},
			},
		},
	}

	return conn.WriteJSON(msg)
}

func (t *OpenAIRealtimeTranscriber) sendLoop(ctx context.Context, conn *websocket.Conn) error {
	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			return nil

		case audio, ok := <-t.audioCh:
			if !ok {
				return nil
			}

			msg := audioBufferAppendMessage{
				Type:  "input_audio_buffer.append",
				Audio: base64.StdEncoding.EncodeToString(audio),
			}

			if err := conn.WriteJSON(msg); err != nil {
				if ctx.Err() == nil {
					t.logger.Error("write audio", "error", err)
				}
				return err
			}
		}
	}
}

func (t *OpenAIRealtimeTranscriber) isReconnectable(err error) bool {
	if err == nil {
		return true
	}

	var sErr *serverError
	if errors.As(err, &sErr) {
		return sErr.ErrorType != "invalid_request_error"
	}

	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		switch closeErr.Code {
		case websocket.CloseNormalClosure,
			websocket.CloseGoingAway,
			websocket.CloseAbnormalClosure,
			websocket.CloseServiceRestart,
			websocket.CloseTryAgainLater:
			return true

		case websocket.ClosePolicyViolation,
			websocket.CloseInvalidFramePayloadData:
			return false

		default:
			return true
		}
	}

	return true
}

func (t *OpenAIRealtimeTranscriber) receiveLoop(ctx context.Context, conn *websocket.Conn) error {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		var evt serverEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			t.logger.Warn("unmarshal event", "error", err)
			continue
		}

		switch evt.Type {
		case "conversation.item.input_audio_transcription.delta":
			if evt.Delta == "" {
				continue
			}

			t.logger.Debug("interim", "text", evt.Delta)

			select {
			case t.transcriptCh <- voice.Transcript{Text: evt.Delta, IsFinal: false}:
			default:
			}

		case "conversation.item.input_audio_transcription.completed":
			if evt.Transcript == "" {
				continue
			}

			t.logger.Info("final", "text", evt.Transcript)

			select {
			case t.transcriptCh <- voice.Transcript{Text: evt.Transcript, IsFinal: true}:
			default:
			}

		case "input_audio_buffer.speech_started":
			t.logger.Debug("speech started")

			select {
			case t.transcriptCh <- voice.Transcript{Text: "", IsFinal: false}:
			default:
			}

		case "input_audio_buffer.speech_stopped":
			t.logger.Debug("speech stopped")

		case "input_audio_buffer.committed":
			t.logger.Debug("audio buffer committed", "item_id", evt.ItemID)

		case "conversation.item.input_audio_transcription.failed":
			if evt.Error != nil {
				t.logger.Error("transcription failed", "item_id", evt.ItemID, "type", evt.Error.Type, "code", evt.Error.Code, "message", evt.Error.Message)
			}

		case "session.created", "session.updated":
			t.logger.Info("session event", "type", evt.Type)

		case "error":
			if evt.Error != nil {
				t.logger.Error("server error", "type", evt.Error.Type, "code", evt.Error.Code, "message", evt.Error.Message)
				return &serverError{
					ErrorType: evt.Error.Type,
					Code:      evt.Error.Code,
					Message:   evt.Error.Message,
				}
			}
			return fmt.Errorf("server error: %s", string(data))

		default:
			t.logger.Debug("unhandled event", "type", evt.Type)
		}
	}
}
