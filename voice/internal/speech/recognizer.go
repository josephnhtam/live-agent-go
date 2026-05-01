package speech

import (
	"context"
	"errors"
	"golang.org/x/sync/errgroup"
	"live-agent-go/voice/core"
)

const bufferSize = 8

type RecognizerConfig struct {
	VAD         VAD
	Transcriber Transcriber
	Handler     RecognitionHandler
}

type Recognizer struct {
	vad         VAD
	transcriber Transcriber
	handler     RecognitionHandler

	ctx    context.Context
	cancel context.CancelFunc
	grp    *errgroup.Group

	ch chan any
}

func NewRecognizer(config RecognizerConfig) *Recognizer {
	ctx, cancel := context.WithCancel(context.Background())

	return &Recognizer{
		vad:         config.VAD,
		transcriber: config.Transcriber,
		handler:     config.Handler,

		ctx:    ctx,
		cancel: cancel,
		ch:     make(chan any, bufferSize),
	}
}

func (r *Recognizer) Start(ctx context.Context) error {
	if err := r.vad.Start(ctx); err != nil {
		return errors.Join(ErrStartingVAD, err)
	}

	if err := r.transcriber.Start(ctx); err != nil {
		return errors.Join(ErrStartingTranscriber, err)
	}

	r.grp, ctx = errgroup.WithContext(r.ctx)

	r.grp.Go(func() error {
		vadCh := r.vad.Event()
		transcribeCh := r.transcriber.Transcribe()

		for {
			select {
			case <-ctx.Done():
				return nil

			case vadEvent, ok := <-vadCh:
				if !ok {
					vadCh = nil
					continue
				}
				select {
				case r.ch <- vadEvent:
				case <-ctx.Done():
					return nil
				}

			case transcript, ok := <-transcribeCh:
				if !ok {
					transcribeCh = nil
					continue
				}
				select {
				case r.ch <- transcript:
				case <-ctx.Done():
					return nil
				}
			}

			if vadCh == nil && transcribeCh == nil {
				return nil
			}
		}
	})

	r.grp.Go(func() error {
		return r.handleEvent(ctx)
	})

	return nil
}

func (r *Recognizer) Stop(ctx context.Context) error {
	r.cancel()

	var errs []error
	errs = append(errs, r.vad.Stop(ctx))
	errs = append(errs, r.transcriber.Stop(ctx))

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- r.grp.Wait()
	}()

	select {
	case <-ctx.Done():
		errs = append(errs, ctx.Err())
	case err := <-waitCh:
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (r *Recognizer) Feed(ctx context.Context, frame core.AudioFrame) error {
	if err := r.vad.Feed(ctx, frame); err != nil {
		return errors.Join(ErrFeedingVAD, err)
	}

	if err := r.transcriber.Feed(ctx, frame); err != nil {
		return errors.Join(ErrFeedingTranscriber, err)
	}

	return nil
}

func (r *Recognizer) handleEvent(ctx context.Context) error {
	isSpeaking := false
	isTranscribing := false
	var transcripts []core.Transcript = nil

	for {
		select {
		case <-ctx.Done():
			return nil

		case evt := <-r.ch:
			switch v := evt.(type) {
			case VADEvent:
				if v == VADEventSpeechStart {
					isSpeaking = true
					r.handler.OnSpeechStart()
				} else if v == VADEventSpeechEnd {
					isSpeaking = false

					if !isTranscribing && len(transcripts) > 0 {
						r.handler.OnSpeechRecognized(transcripts)
						transcripts = nil
					}
				}

			case core.Transcript:
				if v.IsFinal {
					transcripts = append(transcripts, v)
					isTranscribing = false

					if !isSpeaking && len(transcripts) > 0 {
						r.handler.OnSpeechRecognized(transcripts)
						transcripts = nil
					}
				} else {
					isTranscribing = true
				}
			}
		}
	}
}
