package voice

import (
	"context"
	"errors"
	"live-agent-go/voice/internal/dialog"
	"live-agent-go/voice/internal/speech"
	"sync"
	"sync/atomic"
)

type AgentConfig struct {
	VAD         VAD
	Transcriber Transcriber
	Synthesizer Synthesizer
	Brain       Brain
}

type Agent struct {
	config  AgentConfig
	options *agentOptions

	ctx         context.Context
	cancel      context.CancelFunc
	respAudioCh chan<- AudioFrame
	respTokenCh chan<- Token
	respErrCh   chan<- error
	promptCh    chan<- string

	responder          *dialog.Responder
	recognizer         *speech.Recognizer
	recognitionHandler *recognitionHandler

	lock    sync.Mutex
	started atomic.Bool
	stopped atomic.Bool
}

func NewAgent(config AgentConfig, opts ...AgentOption) (*Agent, error) {
	if err := validateAgentConfig(config); err != nil {
		return nil, err
	}

	options := buildAgentOptions(opts...)
	ctx, cancel := context.WithCancel(context.Background())

	return &Agent{
		config:      config,
		options:     options,
		ctx:         ctx,
		cancel:      cancel,
		respAudioCh: options.respAudioCh,
		respTokenCh: options.respTokenCh,
		respErrCh:   options.respErrCh,
		promptCh:    options.promptCh,
		lock:        sync.Mutex{},
		started:     atomic.Bool{},
		stopped:     atomic.Bool{},
	}, nil
}

func (a *Agent) start(ctx context.Context) error {
	a.responder = dialog.NewResponder(dialog.ResponderConfig{
		Brain:       a.config.Brain,
		Synthesizer: a.config.Synthesizer,
		AudioCh:     a.respAudioCh,
		TokenCh:     a.respTokenCh,
		ErrCh:       a.respErrCh,
		PromptCh:    a.promptCh,
	})

	a.recognitionHandler = newRecognitionHandler(recognitionHandlerConfig{
		Ctx:       a.ctx,
		Responder: a.responder,
	})

	a.recognizer = speech.NewRecognizer(speech.RecognizerConfig{
		VAD:         a.config.VAD,
		Transcriber: a.config.Transcriber,
		Handler:     a.recognitionHandler,
	})

	if err := a.recognizer.Start(ctx); err != nil {
		a.recognizer = nil
		a.recognitionHandler = nil
		a.responder = nil
		return errors.Join(ErrStartingRecognizer, err)
	}

	return nil
}

func (a *Agent) stop(ctx context.Context) error {
	a.cancel()
	a.recognitionHandler.CancelResponse()
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
	if config.VAD == nil {
		return ErrInvalidVAD
	}

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
