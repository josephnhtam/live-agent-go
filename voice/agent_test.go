package voice_test

import (
	"context"
	"testing"
	"time"

	"github.com/josephnhtam/live-agent-go/voice/mock_voice"
	"github.com/josephnhtam/live-agent-go/voice"
	"github.com/josephnhtam/live-agent-go/voice/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewAgent_NilTranscriber(t *testing.T) {
	ctrl := gomock.NewController(t)
	brain := mock_voice.NewMockBrain(ctrl)
	synth := mock_voice.NewMockSynthesizer(ctrl)

	_, err := voice.NewAgent(voice.AgentConfig{Brain: brain, Synthesizer: synth}, nil)
	assert.ErrorIs(t, err, voice.ErrInvalidTranscriber)
}

func TestNewAgent_NilBrain(t *testing.T) {
	ctrl := gomock.NewController(t)
	transcriber := mock_voice.NewMockTranscriber(ctrl)
	synth := mock_voice.NewMockSynthesizer(ctrl)

	_, err := voice.NewAgent(voice.AgentConfig{Transcriber: transcriber, Synthesizer: synth}, nil)
	assert.ErrorIs(t, err, voice.ErrInvalidBrain)
}

func TestNewAgent_NilSynthesizer(t *testing.T) {
	ctrl := gomock.NewController(t)
	transcriber := mock_voice.NewMockTranscriber(ctrl)
	brain := mock_voice.NewMockBrain(ctrl)

	_, err := voice.NewAgent(voice.AgentConfig{Transcriber: transcriber, Brain: brain}, nil)
	assert.ErrorIs(t, err, voice.ErrInvalidSynthesizer)
}

func TestNewAgent_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	transcriber := mock_voice.NewMockTranscriber(ctrl)
	brain := mock_voice.NewMockBrain(ctrl)
	synth := mock_voice.NewMockSynthesizer(ctrl)

	agent, err := voice.NewAgent(voice.AgentConfig{
		Transcriber: transcriber,
		Brain:       brain,
		Synthesizer: synth,
	}, nil)
	require.NoError(t, err)
	assert.NotNil(t, agent)
}

func TestAgent_FeedBeforeStart(t *testing.T) {
	ctrl := gomock.NewController(t)
	transcriber := mock_voice.NewMockTranscriber(ctrl)
	brain := mock_voice.NewMockBrain(ctrl)
	synth := mock_voice.NewMockSynthesizer(ctrl)

	agent, err := voice.NewAgent(voice.AgentConfig{
		Transcriber: transcriber,
		Brain:       brain,
		Synthesizer: synth,
	}, nil)
	require.NoError(t, err)

	frame := &core.PCMFrame{PCMData: []int16{1}, SampleRateHz: 16000, NumChannels: 1}
	assert.ErrorIs(t, agent.Feed(context.Background(), frame), voice.ErrNotStarted)
}

func TestAgent_StopBeforeStart(t *testing.T) {
	ctrl := gomock.NewController(t)
	transcriber := mock_voice.NewMockTranscriber(ctrl)
	brain := mock_voice.NewMockBrain(ctrl)
	synth := mock_voice.NewMockSynthesizer(ctrl)

	agent, err := voice.NewAgent(voice.AgentConfig{
		Transcriber: transcriber,
		Brain:       brain,
		Synthesizer: synth,
	}, nil)
	require.NoError(t, err)

	assert.ErrorIs(t, agent.Stop(context.Background()), voice.ErrNotStarted)
}

func TestAgent_StartAndStop(t *testing.T) {
	ctrl := gomock.NewController(t)

	transcribeCh := make(chan voice.Transcript)
	transcriber := mock_voice.NewMockTranscriber(ctrl)
	transcriber.EXPECT().Start(gomock.Any()).Return(nil)
	transcriber.EXPECT().Transcribe().Return(transcribeCh)
	transcriber.EXPECT().Stop(gomock.Any()).Return(nil)

	brain := mock_voice.NewMockBrain(ctrl)
	synth := mock_voice.NewMockSynthesizer(ctrl)
	synth.EXPECT().SampleRate().Return(int32(16000)).AnyTimes()
	synth.EXPECT().Close(gomock.Any()).Return(nil)

	agent, err := voice.NewAgent(voice.AgentConfig{
		Transcriber: transcriber,
		Brain:       brain,
		Synthesizer: synth,
	}, nil)
	require.NoError(t, err)

	require.NoError(t, agent.Start(context.Background()))
	assert.ErrorIs(t, agent.Start(context.Background()), voice.ErrAlreadyStarted)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, agent.Stop(ctx))
	assert.ErrorIs(t, agent.Stop(context.Background()), voice.ErrAlreadyStopped)
}
