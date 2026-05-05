package dialog

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/josephnhtam/live-agent-go/voice/internal/core"

	"github.com/google/uuid"
)

type ResponderConfig struct {
	Ctx                   context.Context
	Brain                 Brain
	Synthesizer           Synthesizer
	BrainBufferSize       int
	SynthesizerBufferSize int
	SynTokenBufferSize    int
	OutputTokenBufferSize int

	AudioChs  []chan<- core.AudioFrame
	TokenChs  []chan<- core.Token
	ErrChs    []chan<- error
	PromptChs []chan<- core.Prompt
	Logger    *slog.Logger
}

type Responder struct {
	ctx                   context.Context
	brain                 Brain
	synthesizer           Synthesizer
	brainBufferSize       int
	synthesizerBufferSize int
	synTokenBufferSize    int
	outputTokenBufferSize int

	audioChs  []chan<- core.AudioFrame
	tokenChs  []chan<- core.Token
	errChs    []chan<- error
	promptChs []chan<- core.Prompt
	logger    *slog.Logger

	mutex      sync.Mutex
	cancelResp context.CancelFunc
	respWg     *sync.WaitGroup
}

func NewResponder(config ResponderConfig) *Responder {
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Responder{
		ctx:                   config.Ctx,
		brain:                 config.Brain,
		synthesizer:           config.Synthesizer,
		brainBufferSize:       bufferSize(config.BrainBufferSize, 32),
		synthesizerBufferSize: bufferSize(config.SynthesizerBufferSize, 32),
		synTokenBufferSize:    bufferSize(config.SynTokenBufferSize, 32),
		outputTokenBufferSize: bufferSize(config.OutputTokenBufferSize, 32),
		audioChs:              config.AudioChs,
		tokenChs:              config.TokenChs,
		errChs:                config.ErrChs,
		promptChs:             config.PromptChs,
		logger:                logger.WithGroup("responder"),
	}
}

func bufferSize(size, fallback int) int {
	if size <= 0 {
		return fallback
	}
	return size
}

func (r *Responder) IceBreaking() {
	r.CancelResponse()

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

	r.CancelResponse()

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

func (r *Responder) CancelResponse() {
	r.mutex.Lock()

	cancel := r.cancelResp
	wg := r.respWg

	r.cancelResp = nil
	r.respWg = nil

	r.mutex.Unlock()

	if cancel != nil {
		cancel()
	}

	if wg != nil {
		wg.Wait()
	}
}

func (r *Responder) createResponseContext() context.Context {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	ctx, cancel := context.WithCancel(r.ctx)
	r.cancelResp = cancel
	return ctx
}

func (r *Responder) generate(ctx context.Context, prompt string, wg *sync.WaitGroup) {
	synIn := make(chan core.Token, r.synTokenBufferSize)
	tokenOut := make(chan core.Token, r.outputTokenBufferSize)
	brainOut := make(chan core.Token, r.brainBufferSize)
	audioOut := make(chan core.AudioFrame, r.synthesizerBufferSize)

	wg.Add(5)

	go func() {
		defer wg.Done()
		defer close(brainOut)
		if err := r.brain.Generate(ctx, prompt, brainOut); err != nil {
			r.sendError(ctx, err)
		}
	}()

	go func() {
		defer wg.Done()
		defer close(audioOut)
		if err := r.synthesizer.Synthesize(ctx, synIn, audioOut); err != nil {
			r.sendError(ctx, err)
		}
	}()

	go func() {
		defer wg.Done()
		r.forwardTokens(ctx, brainOut, synIn, tokenOut)
	}()

	go func() {
		defer wg.Done()
		r.consumeTokens(ctx, tokenOut)
	}()

	go func() {
		defer wg.Done()
		r.consumeAudios(ctx, audioOut)
	}()
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

func (r *Responder) consumeAudios(ctx context.Context, audioOut <-chan core.AudioFrame) {
	if len(r.audioChs) == 0 {
		for range audioOut {
		}
		return
	}

	for audio := range audioOut {
		audio.SetContext(ctx)

		for _, ch := range r.audioChs {
			select {
			case ch <- audio:
			case <-ctx.Done():
			}
		}

		if ctx.Err() != nil {
			break
		}
	}

	for range audioOut {
	}
}
