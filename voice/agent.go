package voice

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/josephnhtam/live-agent-go/voice/helper"
	"github.com/josephnhtam/live-agent-go/voice/internal/dialog"
	"github.com/josephnhtam/live-agent-go/voice/internal/speech"
)

type AgentConfig struct {
	Transcriber Transcriber
	Synthesizer Synthesizer
	Brain       Brain
}

type Agent struct {
	config  AgentConfig
	options *AgentOptions
	logger  *slog.Logger

	ctx          context.Context
	cancel       context.CancelFunc
	respAudioChs []chan<- AudioFrame
	respTokenChs []chan<- Token
	respErrChs   []chan<- error
	promptChs    []chan<- Prompt

	responder          *dialog.Responder
	recognizer         *speech.Recognizer
	recognitionHandler *recognitionHandler

	done    chan error
	lock    sync.Mutex
	started atomic.Bool
	stopped atomic.Bool
}

func NewAgent(config AgentConfig, opts *AgentOptions) (*Agent, error) {
	if err := validateAgentConfig(config); err != nil {
		return nil, err
	}

	options := opts
	if options == nil {
		options = NewAgentOptions()
	}

	logger := options.logger
	if logger == nil {
		logger = helper.NoopLogger()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Agent{
		config:       config,
		options:      options,
		logger:       logger.WithGroup("agent"),
		ctx:          ctx,
		cancel:       cancel,
		respAudioChs: options.respAudioChs,
		respTokenChs: options.respTokenChs,
		respErrChs:   options.respErrChs,
		promptChs:    options.promptChs,
		lock:         sync.Mutex{},
		started:      atomic.Bool{},
		stopped:      atomic.Bool{},
	}, nil
}

func (a *Agent) start(ctx context.Context) error {
	a.responder = dialog.NewResponder(dialog.ResponderConfig{
		Ctx:                   a.ctx,
		Brain:                 a.config.Brain,
		Synthesizer:           a.config.Synthesizer,
		BrainBufferSize:       a.options.brainBufferSize,
		SynthesizerBufferSize: a.options.synthesizerBufferSize,
		SynTokenBufferSize:    a.options.synTokenBufferSize,
		OutputTokenBufferSize: a.options.outputTokenBufferSize,
		AudioChs:              a.respAudioChs,
		TokenChs:              a.respTokenChs,
		ErrChs:                a.respErrChs,
		PromptChs:             a.promptChs,
		Logger:                a.logger,
	})

	a.recognitionHandler = newRecognitionHandler(recognitionHandlerConfig{
		Responder:            a.responder,
		MinInterruptDuration: a.options.minInterruptDuration,
		InterruptOnInterim:   a.options.interruptOnInterim,
	})

	a.recognizer = speech.NewRecognizer(speech.RecognizerConfig{
		VAD:         a.options.vad,
		Transcriber: a.config.Transcriber,
		Handler:     a.recognitionHandler,
	})

	if err := a.recognizer.Start(ctx); err != nil {
		a.recognizer = nil
		a.recognitionHandler = nil
		a.responder = nil
		return errors.Join(ErrStartingRecognizer, err)
	}

	a.done = make(chan error, 1)
	go func() {
		select {
		case err := <-a.recognizer.Done():
			a.done <- err
		case <-a.ctx.Done():
			a.done <- a.ctx.Err()
		}
	}()

	return nil
}

func (a *Agent) IceBreaking() {
	a.responder.IceBreaking()
}

func (a *Agent) Done() <-chan error {
	return a.done
}

func (a *Agent) stop(ctx context.Context) error {
	a.cancel()
	a.responder.CancelResponse()
	return a.recognizer.Stop(ctx)
}

func (a *Agent) Feed(ctx context.Context, frame AudioFrame) error {
	if !a.started.Load() {
		return ErrNotStarted
	}

	if a.stopped.Load() {
		return ErrAlreadyStopped
	}

	return a.recognizer.Feed(ctx, frame)
}

func (a *Agent) Start(ctx context.Context) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.started.Load() {
		return ErrAlreadyStarted
	}

	if err := a.start(ctx); err != nil {
		return err
	}

	a.started.Store(true)
	return nil
}

func (a *Agent) Stop(ctx context.Context) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if !a.started.Load() {
		return ErrNotStarted
	}

	if a.stopped.Load() {
		return ErrAlreadyStopped
	}

	if err := a.stop(ctx); err != nil {
		return err
	}

	a.stopped.Store(true)
	return nil
}

func validateAgentConfig(config AgentConfig) error {
	if config.Transcriber == nil {
		return ErrInvalidTranscriber
	}

	if config.Brain == nil {
		return ErrInvalidBrain
	}

	if config.Synthesizer == nil {
		return ErrInvalidSynthesizer
	}

	return nil
}
