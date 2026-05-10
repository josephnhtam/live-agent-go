package dialog

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/josephnhtam/live-agent-go/voice/helper"
	"github.com/josephnhtam/live-agent-go/voice/internal/core"
)

type ResponderConfig struct {
	Brain                 Brain
	Synthesizer           Synthesizer
	BrainBufferSize       int
	MixerOutBufferSize    int
	SynthOutBufferSize    int
	SynthInBufferSize     int
	OutputTokenBufferSize int

	AudioChs  []chan<- core.AudioFrame
	TokenChs  []chan<- core.Token
	ErrChs    []chan<- error
	PromptChs []chan<- core.Prompt
	Logger    *slog.Logger
}

type Responder struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	brain                 Brain
	synthesizer           Synthesizer
	brainBufferSize       int
	mixerOutBufferSize    int
	synthOutBufferSize    int
	synthInBufferSize     int
	outputTokenBufferSize int

	audioChs  []chan<- core.AudioFrame
	tokenChs  []chan<- core.Token
	errChs    []chan<- error
	promptChs []chan<- core.Prompt
	mixer     *mixer
	logger    *slog.Logger

	interruptible atomic.Bool

	mutex      sync.Mutex
	cancelResp context.CancelFunc
	wg         *sync.WaitGroup
	respWg     *sync.WaitGroup
}

func NewResponder(config ResponderConfig) *Responder {
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	mixerOut := make(chan core.AudioFrame, config.MixerOutBufferSize)
	mixer := newMixer(mixerOut, config.Synthesizer.SampleRate(), logger)

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	r := &Responder{
		ctx:                   ctx,
		cancel:                cancel,
		brain:                 config.Brain,
		synthesizer:           config.Synthesizer,
		brainBufferSize:       config.BrainBufferSize,
		synthOutBufferSize:    config.SynthOutBufferSize,
		synthInBufferSize:     config.SynthInBufferSize,
		outputTokenBufferSize: config.OutputTokenBufferSize,
		audioChs:              config.AudioChs,
		tokenChs:              config.TokenChs,
		errChs:                config.ErrChs,
		promptChs:             config.PromptChs,
		mixer:                 mixer,
		logger:                logger.WithGroup("responder"),
		wg:                    wg,
	}

	r.interruptible.Store(true)

	wg.Add(2)

	go func() {
		defer wg.Done()
		mixer.Run()
	}()

	go func() {
		defer wg.Done()
		r.consumeAudios(mixerOut)
	}()

	return r
}

func (r *Responder) Close(ctx context.Context) error {
	r.cancel()

	return errors.Join(
		r.CancelResponse(ctx),
		r.mixer.Close(ctx),
		r.synthesizer.Close(ctx),
		helper.WaitWithCtx(ctx, r.wg),
	)
}

func (r *Responder) SetInterruptible(interruptible bool) {
	r.interruptible.Store(interruptible)
}

func (r *Responder) IsInterruptible() bool {
	return r.interruptible.Load()
}

func (r *Responder) IceBreaking() {
	r.CancelResponse(context.Background())

	ctx := r.createResponseContext()
	wg := &sync.WaitGroup{}
	r.generate(ctx, "", wg)

	r.mutex.Lock()
	r.respWg = wg
	r.mutex.Unlock()
}

func (r *Responder) Respond(prompt string) {
	if strings.TrimSpace(prompt) == "" {
		return
	}

	r.CancelResponse(context.Background())

	ctx := r.createResponseContext()
	wg := &sync.WaitGroup{}

	for _, ch := range r.promptChs {
		select {
		case ch <- core.Prompt{MessageID: uuid.NewString(), Text: prompt}:
		case <-ctx.Done():
			return
		}
	}

	r.generate(ctx, prompt, wg)

	r.mutex.Lock()
	r.respWg = wg
	r.mutex.Unlock()
}

func (r *Responder) CancelResponse(ctx context.Context) error {
	r.mutex.Lock()

	cancel := r.cancelResp
	wg := r.respWg

	r.cancelResp = nil
	r.respWg = nil

	r.mutex.Unlock()

	if cancel != nil {
		cancel()
		r.mixer.SetSpeechSource(nil)
	}

	if wg != nil {
		return helper.WaitWithCtx(ctx, wg)
	}

	return nil
}

func (r *Responder) createResponseContext() context.Context {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	r.cancelResp = cancel
	return ctx
}

func (r *Responder) generate(ctx context.Context, prompt string, wg *sync.WaitGroup) {
	r.interruptible.Store(true)

	synthIn := make(chan core.Token, r.synthInBufferSize)
	tokenOut := make(chan core.Token, r.outputTokenBufferSize)
	brainOut := make(chan core.Token, r.brainBufferSize)
	synthOut := make(chan core.AudioFrame, r.synthOutBufferSize)

	wg.Add(5)

	go func() {
		defer wg.Done()
		defer close(brainOut)

		tools := newTools(brainOut, r.mixer, r)

		if err := r.brain.Generate(ctx, prompt, tools, brainOut); err != nil {
			r.sendError(ctx, err)
		}
	}()

	go func() {
		defer wg.Done()
		defer close(synthOut)
		defer r.interruptible.Store(true)

		if err := r.synthesizer.Synthesize(ctx, synthIn, synthOut); err != nil {
			r.sendError(ctx, err)
		}
	}()

	go func() {
		defer wg.Done()
		r.forwardSynthToMixer(ctx, synthOut)
	}()

	go func() {
		defer wg.Done()
		r.forwardTokens(ctx, brainOut, synthIn, tokenOut)
	}()

	go func() {
		defer wg.Done()
		r.consumeTokens(ctx, tokenOut)
	}()
}

func (r *Responder) forwardSynthToMixer(ctx context.Context, synthOut <-chan core.AudioFrame) {
	mixerIn := make(chan core.AudioFrame, r.synthOutBufferSize)
	first := true

	defer func() {
		if !first {
			close(mixerIn)
		}
	}()

	for frame := range synthOut {
		if ctx.Err() != nil {
			return
		}

		if first {
			r.mixer.SetSpeechSource(mixerIn)
			first = false
		}

		select {
		case mixerIn <- frame:
		case <-ctx.Done():
			return
		}
	}
}

func (r *Responder) forwardTokens(ctx context.Context,
	brainOut <-chan core.Token, synIn chan<- core.Token, tokenOut chan<- core.Token) {
	defer close(synIn)
	defer close(tokenOut)

	for token := range brainOut {
		if ctx.Err() != nil {
			return
		}

		select {
		case <-ctx.Done():
			return
		case tokenOut <- token:
		default:
			r.logger.Warn("token output channel full, dropping token")
		}

		select {
		case <-ctx.Done():
			return
		case synIn <- token:
		default:
			r.logger.Warn("synthesizer token channel full, dropping token")
		}
	}
}

func (r *Responder) sendError(ctx context.Context, err error) {
	for _, ch := range r.errChs {
		select {
		case ch <- err:
		case <-ctx.Done():
			return
		}
	}
}

func (r *Responder) consumeTokens(ctx context.Context, tokenOut <-chan core.Token) {
	if len(r.tokenChs) == 0 {
		for range tokenOut {
		}
		return
	}

	for token := range tokenOut {
		for _, ch := range r.tokenChs {
			select {
			case ch <- token:
			case <-ctx.Done():
			}
		}

		if ctx.Err() != nil {
			break
		}
	}

	for range tokenOut {
	}
}

func (r *Responder) consumeAudios(mixerOut <-chan core.AudioFrame) {
	if len(r.audioChs) == 0 {
		for range mixerOut {
		}
		return
	}

	for audio := range mixerOut {
		for _, ch := range r.audioChs {
			if r.ctx.Err() != nil {
				return
			}

			select {
			case ch <- audio:
			case <-r.ctx.Done():
				return
			}
		}
	}
}
