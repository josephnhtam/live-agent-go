package synthesizer

import (
	"context"
	"fmt"
	"github.com/josephnhtam/live-agent-go/voice"
	"github.com/josephnhtam/live-agent-go/voice/helper"
	"log/slog"
	"strings"
	"sync"

	msginterfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/api/speak/v1/websocket/interfaces"
	interfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/interfaces/v1"
	speakClient "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/speak"
)

type DeepgramSynthesizer struct {
	config  DeepgramSynthesizerConfig
	options DeepgramSynthesizerOptions
	logger  *slog.Logger

	mutex    sync.Mutex
	client   *speakClient.WSCallback
	cancel   context.CancelFunc
	callback *deepgramTTSCallback
}

var _ voice.Synthesizer = (*DeepgramSynthesizer)(nil)

func NewDeepgramSynthesizer(config DeepgramSynthesizerConfig, opts *DeepgramSynthesizerOptions) *DeepgramSynthesizer {
	options := opts
	if options == nil {
		options = NewDeepgramSynthesizerOptions()
	}

	logger := options.logger
	if logger == nil {
		logger = helper.NoopLogger()
	}

	return &DeepgramSynthesizer{
		config:  config,
		options: *options,
		logger:  logger.WithGroup("deepgram_tts"),
	}
}

func (s *DeepgramSynthesizer) SampleRate() int32 {
	return s.options.sampleRate
}

func (s *DeepgramSynthesizer) Synthesize(ctx context.Context, tokens <-chan voice.Token, audioCh chan<- voice.AudioFrame) error {
	if err := s.ensureConnected(ctx); err != nil {
		return err
	}

	s.callback.setSession(ctx, audioCh)
	defer func() {
		if s.callback != nil {
			s.callback.clearSession()
		}
	}()

	s.sendLoop(ctx, tokens, audioCh)

	if ctx.Err() != nil {
		s.clearClient()
	}

	return nil
}

func (s *DeepgramSynthesizer) getClient() *speakClient.WSCallback {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.client
}

func (s *DeepgramSynthesizer) Close(_ context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.client != nil {
		s.client.Stop()
		s.client = nil
	}

	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}

	return nil
}

func (s *DeepgramSynthesizer) ensureConnected(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.client != nil {
		return nil
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	connCtx, cancel := context.WithCancel(context.Background())

	callback := &deepgramTTSCallback{
		synth:     s,
		flushedCh: make(chan struct{}, 16),
	}

	clientOptions := &interfaces.ClientOptions{
		APIKey: s.config.APIKey,
	}

	speakOptions := &interfaces.WSSpeakOptions{
		Model:      s.config.Model,
		Encoding:   "linear16",
		SampleRate: int(s.options.sampleRate),
	}

	client, err := speakClient.NewWSUsingCallbackWithCancel(connCtx, cancel, s.config.APIKey, clientOptions, speakOptions, callback)

	if err != nil {
		cancel()
		return fmt.Errorf("%w: %w", ErrCreateClient, err)
	}

	if !client.Connect() {
		cancel()
		return fmt.Errorf("%w: deepgram speak websocket", ErrOpenStream)
	}

	s.client = client
	s.cancel = cancel
	s.callback = callback

	return nil
}

func (s *DeepgramSynthesizer) sendLoop(ctx context.Context, tokens <-chan voice.Token, audioCh chan<- voice.AudioFrame) {
	sb := strings.Builder{}

	for {
		if ctx.Err() != nil {
			return
		}

		select {
		case <-ctx.Done():
			return

		case token, ok := <-tokens:
			if !ok {
				if ctx.Err() != nil {
					return
				}

				s.flushText(ctx, &sb, audioCh)
				s.waitForFlushed(ctx)
				return
			}

			sb.WriteString(token.Text)
			s.trySendSentence(ctx, &sb, audioCh)
		}
	}
}

func (s *DeepgramSynthesizer) clearClient() {
	s.mutex.Lock()
	client := s.client
	cancel := s.cancel
	s.client = nil
	s.cancel = nil
	s.callback = nil
	s.mutex.Unlock()

	if client != nil {
		client.Stop()
	}
	if cancel != nil {
		cancel()
	}
}

func (s *DeepgramSynthesizer) reconnect(ctx context.Context, audioCh chan<- voice.AudioFrame) bool {
	if err := s.ensureConnected(ctx); err != nil {
		s.logger.Error("reconnect failed", "error", err)
		return false
	}

	s.callback.setSession(ctx, audioCh)
	s.logger.Info("reconnected")
	return true
}

func (s *DeepgramSynthesizer) trySendSentence(ctx context.Context, sb *strings.Builder, audioCh chan<- voice.AudioFrame) {
	before, after, found := helper.SplitAtSentenceEnd(sb.String(), s.options.sentenceEndRunes)
	if !found {
		return
	}

	s.speakAndFlush(ctx, before, audioCh)
	sb.Reset()
	sb.WriteString(after)
}

func (s *DeepgramSynthesizer) flushText(ctx context.Context, sb *strings.Builder, audioCh chan<- voice.AudioFrame) {
	if sb.Len() == 0 {
		return
	}

	s.speakAndFlush(ctx, sb.String(), audioCh)
	sb.Reset()
}

func (s *DeepgramSynthesizer) speakAndFlush(ctx context.Context, text string, audioCh chan<- voice.AudioFrame) {
	s.logger.Info("sending text", "text", text)

	client := s.getClient()
	if client == nil {
		if !s.reconnect(ctx, audioCh) {
			return
		}
		client = s.getClient()
	}

	if err := client.SpeakWithText(text); err != nil {
		s.logger.Error("speak text", "error", err)
		if !s.reconnect(ctx, audioCh) {
			return
		}
		client = s.getClient()
		if client == nil {
			return
		}
		if err := client.SpeakWithText(text); err != nil {
			s.logger.Error("speak text after reconnect", "error", err)
			return
		}
	}

	if err := client.Flush(); err != nil {
		s.logger.Error("flush", "error", err)
	} else {
		s.callback.pendingFlushes++
	}
}

func (s *DeepgramSynthesizer) waitForFlushed(ctx context.Context) {
	for s.callback.pendingFlushes > 0 {
		select {
		case <-s.callback.flushedCh:
			s.callback.pendingFlushes--
		case <-ctx.Done():
			return
		}
	}
}

type deepgramTTSCallback struct {
	synth          *DeepgramSynthesizer
	flushedCh      chan struct{}
	pendingFlushes int

	mutex   sync.RWMutex
	ctx     context.Context
	audioCh chan<- voice.AudioFrame
}

var _ msginterfaces.SpeakMessageCallback = (*deepgramTTSCallback)(nil)

func (c *deepgramTTSCallback) setSession(ctx context.Context, audioCh chan<- voice.AudioFrame) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for len(c.flushedCh) > 0 {
		<-c.flushedCh
	}

	c.pendingFlushes = 0
	c.ctx = ctx
	c.audioCh = audioCh
}

func (c *deepgramTTSCallback) clearSession() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.ctx = nil
	c.audioCh = nil
}

func (c *deepgramTTSCallback) Binary(byMsg []byte) error {
	c.mutex.RLock()
	ctx := c.ctx
	audioCh := c.audioCh
	c.mutex.RUnlock()

	if audioCh == nil || ctx == nil || ctx.Err() != nil {
		return nil
	}

	if len(byMsg) == 0 {
		return nil
	}

	samples := helper.BytesToInt16s(byMsg)
	select {
	case audioCh <- &voice.PCMFrame{
		PCMData:      samples,
		SampleRateHz: c.synth.options.sampleRate,
		NumChannels:  1,
		Ctx:          ctx,
	}:
	case <-ctx.Done():
	}

	return nil
}

func (c *deepgramTTSCallback) Flush(_ *msginterfaces.FlushedResponse) error {
	select {
	case c.flushedCh <- struct{}{}:
	default:
	}
	return nil
}

func (c *deepgramTTSCallback) Open(_ *msginterfaces.OpenResponse) error {
	c.synth.logger.Info("connection opened")
	return nil
}

func (c *deepgramTTSCallback) Metadata(_ *msginterfaces.MetadataResponse) error {
	return nil
}

func (c *deepgramTTSCallback) Clear(_ *msginterfaces.ClearedResponse) error {
	return nil
}

func (c *deepgramTTSCallback) Close(_ *msginterfaces.CloseResponse) error {
	c.synth.logger.Info("connection closed")

	c.synth.mutex.Lock()
	defer c.synth.mutex.Unlock()
	if c.synth.callback == c {
		c.synth.client = nil
		c.synth.cancel = nil
		c.synth.callback = nil
	}

	return nil
}

func (c *deepgramTTSCallback) Warning(wr *msginterfaces.WarningResponse) error {
	c.synth.logger.Warn("deepgram warning", "code", wr.WarnCode, "message", wr.WarnMsg)
	return nil
}

func (c *deepgramTTSCallback) Error(er *msginterfaces.ErrorResponse) error {
	c.synth.logger.Error("deepgram error", "code", er.ErrCode, "message", er.ErrMsg)
	return nil
}

func (c *deepgramTTSCallback) UnhandledEvent(byMsg []byte) error {
	c.synth.logger.Warn("unhandled event", "data", string(byMsg))
	return nil
}
