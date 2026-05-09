package dialog_test

import (
	"context"
	"testing"
	"time"

	"github.com/josephnhtam/live-agent-go/voice/internal/dialog/mock_dialog"
	"github.com/josephnhtam/live-agent-go/voice/helper"
	"github.com/josephnhtam/live-agent-go/voice/internal/core"
	"github.com/josephnhtam/live-agent-go/voice/internal/dialog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestResponder(t *testing.T, ctrl *gomock.Controller) (*dialog.Responder, *mock_dialog.MockBrain, *mock_dialog.MockSynthesizer, chan core.Token, chan core.AudioFrame) {
	mockBrain := mock_dialog.NewMockBrain(ctrl)
	mockSynth := mock_dialog.NewMockSynthesizer(ctrl)
	mockSynth.EXPECT().SampleRate().Return(int32(16000)).AnyTimes()

	tokenCh := make(chan core.Token, 32)
	audioCh := make(chan core.AudioFrame, 32)

	r := dialog.NewResponder(dialog.ResponderConfig{
		Brain:                 mockBrain,
		Synthesizer:           mockSynth,
		BrainBufferSize:       8,
		MixerOutBufferSize:    8,
		SynthOutBufferSize:    8,
		SynthInBufferSize:     8,
		OutputTokenBufferSize: 8,
		TokenChs:              []chan<- core.Token{tokenCh},
		AudioChs:              []chan<- core.AudioFrame{audioCh},
		Logger:                helper.NoopLogger(),
	})

	return r, mockBrain, mockSynth, tokenCh, audioCh
}

func TestResponder_Interruptible(t *testing.T) {
	ctrl := gomock.NewController(t)
	r, _, _, _, _ := newTestResponder(t, ctrl)
	defer r.Close(context.Background())

	assert.True(t, r.IsInterruptible())

	r.SetInterruptible(false)
	assert.False(t, r.IsInterruptible())

	r.SetInterruptible(true)
	assert.True(t, r.IsInterruptible())
}

func TestResponder_Respond_GeneratesTokens(t *testing.T) {
	ctrl := gomock.NewController(t)
	r, mockBrain, mockSynth, tokenCh, _ := newTestResponder(t, ctrl)
	defer r.Close(context.Background())

	mockBrain.EXPECT().Generate(gomock.Any(), "hello", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, prompt string, tools dialog.Tools, tokens chan<- core.Token) error {
			tokens <- core.Token{Text: "response"}
			return nil
		})

	mockSynth.EXPECT().Synthesize(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tokens <-chan core.Token, audio chan<- core.AudioFrame) error {
			for range tokens {
			}
			return nil
		})

	r.Respond("hello")

	select {
	case tok := <-tokenCh:
		assert.Equal(t, "response", tok.Text)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for token")
	}
}

func TestResponder_CancelResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	r, _, _, _, _ := newTestResponder(t, ctrl)
	defer r.Close(context.Background())

	assert.NoError(t, r.CancelResponse(context.Background()))
}

func TestResponder_IceBreaking(t *testing.T) {
	ctrl := gomock.NewController(t)
	r, mockBrain, mockSynth, _, _ := newTestResponder(t, ctrl)
	defer r.Close(context.Background())

	brainCalled := make(chan struct{})
	mockBrain.EXPECT().Generate(gomock.Any(), "", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, prompt string, tools dialog.Tools, tokens chan<- core.Token) error {
			close(brainCalled)
			return nil
		})

	mockSynth.EXPECT().Synthesize(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tokens <-chan core.Token, audio chan<- core.AudioFrame) error {
			for range tokens {
			}
			return nil
		})

	r.IceBreaking()

	select {
	case <-brainCalled:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for brain.Generate call")
	}
}

func TestResponder_Close(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockBrain := mock_dialog.NewMockBrain(ctrl)
	mockSynth := mock_dialog.NewMockSynthesizer(ctrl)
	mockSynth.EXPECT().SampleRate().Return(int32(16000)).AnyTimes()

	r := dialog.NewResponder(dialog.ResponderConfig{
		Brain:                 mockBrain,
		Synthesizer:           mockSynth,
		BrainBufferSize:       8,
		MixerOutBufferSize:    8,
		SynthOutBufferSize:    8,
		SynthInBufferSize:     8,
		OutputTokenBufferSize: 8,
		Logger:                helper.NoopLogger(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	require.NoError(t, r.Close(ctx))
}

func TestResponder_Respond_EmptyPromptIgnored(t *testing.T) {
	ctrl := gomock.NewController(t)
	r, _, _, _, _ := newTestResponder(t, ctrl)
	defer r.Close(context.Background())

	r.Respond("  ")
}
