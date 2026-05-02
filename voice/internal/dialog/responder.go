package dialog

import (
	"context"
	"live-agent-go/voice/core"
	"sync"
)

type ResponderConfig struct {
	Brain       Brain
	Synthesizer Synthesizer

	AudioCh  chan<- core.AudioFrame
	TokenCh  chan<- core.Token
	ErrCh    chan<- error
	PromptCh chan<- string
}

type Responder struct {
	brain       Brain
	synthesizer Synthesizer

	audioCh  chan<- core.AudioFrame
	tokenCh  chan<- core.Token
	errCh    chan<- error
	promptCh chan<- string
}

func NewResponder(config ResponderConfig) *Responder {
	return &Responder{
		brain:       config.Brain,
		synthesizer: config.Synthesizer,
		audioCh:     config.AudioCh,
		tokenCh:     config.TokenCh,
		errCh:       config.ErrCh,
		promptCh:    config.PromptCh,
	}
}

func (r *Responder) Respond(ctx context.Context, prompt string) *sync.WaitGroup {
	wg := &sync.WaitGroup{}

	if r.promptCh != nil {
		select {
		case r.promptCh <- prompt:
		case <-ctx.Done():
			return wg
		}
	}

	synInputCh := make(chan core.Token)
	outputTokenCh := make(chan core.Token)

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
		case synInputCh <- token:
		}

		select {
		case <-ctx.Done():
			return
		case outputTokenCh <- token:
		}
	}
}

func (r *Responder) sendError(ctx context.Context, err error) {
	if r.errCh == nil {
		return
	}

	select {
	case r.errCh <- err:
	case <-ctx.Done():
	}
}

func (r *Responder) consumeTokens(ctx context.Context, outputTokenCh <-chan core.Token) {
	if r.tokenCh == nil {
		for range outputTokenCh {
		}
		return
	}

	for token := range outputTokenCh {
		select {
		case r.tokenCh <- token:
		case <-ctx.Done():
		}

		if ctx.Err() != nil {
			break
		}
	}

	for range outputTokenCh {
	}
}

func (r *Responder) consumeAudios(ctx context.Context, outputAudioCh <-chan core.AudioFrame) {
	if r.audioCh == nil {
		for range outputAudioCh {
		}
		return
	}

	for audio := range outputAudioCh {
		select {
		case r.audioCh <- audio:
		case <-ctx.Done():
		}

		if ctx.Err() != nil {
			break
		}
	}

	for range outputAudioCh {
	}
}
