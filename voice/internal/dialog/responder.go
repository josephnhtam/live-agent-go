package dialog

import (
	"context"
	"live-agent-go/voice/core"
	"sync"
)

type ResponderConfig struct {
	Brain       Brain
	Synthesizer Synthesizer

	AudioChs  []chan<- core.AudioFrame
	TokenChs  []chan<- core.Token
	ErrChs    []chan<- error
	PromptChs []chan<- string
}

type Responder struct {
	brain       Brain
	synthesizer Synthesizer

	audioChs  []chan<- core.AudioFrame
	tokenChs  []chan<- core.Token
	errChs    []chan<- error
	promptChs []chan<- string
}

func NewResponder(config ResponderConfig) *Responder {
	return &Responder{
		brain:       config.Brain,
		synthesizer: config.Synthesizer,
		audioChs:    config.AudioChs,
		tokenChs:    config.TokenChs,
		errChs:      config.ErrChs,
		promptChs:   config.PromptChs,
	}
}

func (r *Responder) Respond(ctx context.Context, prompt string) *sync.WaitGroup {
	wg := &sync.WaitGroup{}

	for _, ch := range r.promptChs {
		select {
		case ch <- prompt:
		case <-ctx.Done():
			return wg
		}
	}

	synInputCh := make(chan core.Token, 32)
	outputTokenCh := make(chan core.Token, 32)

	inputTokenCh, err := r.brain.Generate(ctx, prompt)
	if err != nil {
		r.sendError(ctx, err)
		return wg
	}

	outputAudioCh, err := r.synthesizer.Synthesize(ctx, synInputCh)
	if err != nil {
		r.sendError(ctx, err)
		return wg
	}

	wg.Add(3)

	go func() {
		defer wg.Done()
		r.forwardTokens(ctx, inputTokenCh, synInputCh, outputTokenCh)
	}()

	go func() {
		defer wg.Done()
		r.consumeTokens(ctx, outputTokenCh)
	}()

	go func() {
		defer wg.Done()
		r.consumeAudios(ctx, outputAudioCh)
	}()

	return wg
}

func (r *Responder) forwardTokens(ctx context.Context,
	inputTokenCh <-chan core.Token, synInputCh chan<- core.Token, outputTokenCh chan<- core.Token) {
	defer close(synInputCh)
	defer close(outputTokenCh)

	for token := range inputTokenCh {
		if ctx.Err() != nil {
			return
		}

		select {
		case <-ctx.Done():
			return
		case outputTokenCh <- token:
		}

		select {
		case synInputCh <- token:
		default:
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

func (r *Responder) consumeTokens(ctx context.Context, outputTokenCh <-chan core.Token) {
	if len(r.tokenChs) == 0 {
		for range outputTokenCh {
		}
		return
	}

	for token := range outputTokenCh {
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

	for range outputTokenCh {
	}
}

func (r *Responder) consumeAudios(ctx context.Context, outputAudioCh <-chan core.AudioFrame) {
	if len(r.audioChs) == 0 {
		for range outputAudioCh {
		}
		return
	}

	for audio := range outputAudioCh {
		audio.Ctx = ctx

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

	for range outputAudioCh {
	}
}
