package voice

import (
	"context"
	"errors"
	"live-agent-go/voice/internal/dialog"
	"live-agent-go/voice/internal/speech"
	"sync"
	"sync/atomic"
)

type AgentSetup struct {
	VAD         VAD
	Transcriber Transcriber
	Synthesizer Synthesizer
	Brain       Brain
}

type Agent struct {
	setup   AgentSetup
	options *agentOptions

	ctx         context.Context
	cancel      context.CancelFunc
	respAudioCh chan<- AudioFrame
	respTokenCh chan<- Token
	respErrCh   chan<- error

	responder          *dialog.Responder
	recognizer         *speech.Recognizer
	recognitionHandler *recognitionHandler

	lock    sync.Mutex
	started atomic.Bool
	stopped atomic.Bool
}

func NewAgent(setup AgentSetup, opts ...AgentOption) (*Agent, error) {
	if err := validateAgentSetup(setup); err != nil {
		return nil, err
	}

	options := buildAgentOptions(opts...)
	ctx, cancel := context.WithCancel(context.Background())

	return &Agent{
		setup:       setup,
		options:     options,
		ctx:         ctx,
		cancel:      cancel,
		respAudioCh: options.respAudioCh,
		respTokenCh: options.respTokenCh,
		respErrCh:   options.respErrCh,
		lock:        sync.Mutex{},
		started:     atomic.Bool{},
		stopped:     atomic.Bool{},
	}, nil
}

func (a *Agent) start(ctx context.Context) error {
	a.responder = dialog.NewResponder(dialog.ResponderOptions{
		Brain:       a.setup.Brain,
		Synthesizer: a.setup.Synthesizer,
		AudioCh:     a.respAudioCh,
		TokenCh:     a.respTokenCh,
		ErrCh:       a.respErrCh,
	})

	a.recognitionHandler = newRecognitionHandler(recognitionHandlerOptions{
		Responder: a.responder,
	})

	a.recognizer = speech.NewRecognizer(speech.RecognizerOptions{
		VAD:         a.setup.VAD,
		Transcriber: a.setup.Transcriber,
		Handler:     a.recognitionHandler,
	})

	if err := a.recognizer.Start(ctx); err != nil {
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

func validateAgentSetup(options AgentSetup) error {
	if options.VAD == nil {
		return ErrInvalidVAD
	}

	if options.Transcriber == nil {
		return ErrInvalidTranscriber
	}

	if options.Brain == nil {
		return ErrInvalidBrain
	}

	if options.Synthesizer == nil {
		return ErrInvalidSynthesizer
	}

	return nil
}
