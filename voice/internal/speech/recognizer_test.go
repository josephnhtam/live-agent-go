package speech_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/josephnhtam/live-agent-go/voice/internal/speech/mock_speech"
	"github.com/josephnhtam/live-agent-go/voice/internal/core"
	"github.com/josephnhtam/live-agent-go/voice/internal/speech"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRecognizer_Feed_VADError(t *testing.T) {
	ctrl := gomock.NewController(t)

	vadErr := errors.New("vad feed error")
	mockVAD := mock_speech.NewMockVAD(ctrl)
	mockTranscriber := mock_speech.NewMockTranscriber(ctrl)
	mockHandler := mock_speech.NewMockRecognitionHandler(ctrl)

	frame := &core.PCMFrame{PCMData: []int16{1}, SampleRateHz: 16000, NumChannels: 1}
	mockVAD.EXPECT().Feed(gomock.Any(), frame).Return(vadErr)

	r := speech.NewRecognizer(speech.RecognizerConfig{
		VAD:         mockVAD,
		Transcriber: mockTranscriber,
		Handler:     mockHandler,
	})

	err := r.Feed(context.Background(), frame)
	assert.ErrorIs(t, err, speech.ErrFeedingVAD)
	assert.ErrorIs(t, err, vadErr)
}

func TestRecognizer_Feed_TranscriberError(t *testing.T) {
	ctrl := gomock.NewController(t)

	tErr := errors.New("transcriber feed error")
	mockVAD := mock_speech.NewMockVAD(ctrl)
	mockTranscriber := mock_speech.NewMockTranscriber(ctrl)
	mockHandler := mock_speech.NewMockRecognitionHandler(ctrl)

	frame := &core.PCMFrame{PCMData: []int16{1}, SampleRateHz: 16000, NumChannels: 1}
	mockVAD.EXPECT().Feed(gomock.Any(), frame).Return(nil)
	mockTranscriber.EXPECT().Feed(gomock.Any(), frame).Return(tErr)

	r := speech.NewRecognizer(speech.RecognizerConfig{
		VAD:         mockVAD,
		Transcriber: mockTranscriber,
		Handler:     mockHandler,
	})

	err := r.Feed(context.Background(), frame)
	assert.ErrorIs(t, err, speech.ErrFeedingTranscriber)
	assert.ErrorIs(t, err, tErr)
}

func TestRecognizer_Feed_NoVAD(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockTranscriber := mock_speech.NewMockTranscriber(ctrl)
	mockHandler := mock_speech.NewMockRecognitionHandler(ctrl)

	frame := &core.PCMFrame{PCMData: []int16{1}, SampleRateHz: 16000, NumChannels: 1}
	mockTranscriber.EXPECT().Feed(gomock.Any(), frame).Return(nil)

	r := speech.NewRecognizer(speech.RecognizerConfig{
		Transcriber: mockTranscriber,
		Handler:     mockHandler,
	})

	assert.NoError(t, r.Feed(context.Background(), frame))
}

func TestRecognizer_Start_VADError(t *testing.T) {
	ctrl := gomock.NewController(t)

	vadErr := errors.New("vad start error")
	mockVAD := mock_speech.NewMockVAD(ctrl)
	mockTranscriber := mock_speech.NewMockTranscriber(ctrl)
	mockHandler := mock_speech.NewMockRecognitionHandler(ctrl)

	mockVAD.EXPECT().Start(gomock.Any()).Return(vadErr)

	r := speech.NewRecognizer(speech.RecognizerConfig{
		VAD:         mockVAD,
		Transcriber: mockTranscriber,
		Handler:     mockHandler,
	})

	err := r.Start(context.Background())
	assert.ErrorIs(t, err, speech.ErrStartingVAD)
	assert.ErrorIs(t, err, vadErr)
}

func TestRecognizer_Start_TranscriberError(t *testing.T) {
	ctrl := gomock.NewController(t)

	tErr := errors.New("transcriber start error")
	mockVAD := mock_speech.NewMockVAD(ctrl)
	mockTranscriber := mock_speech.NewMockTranscriber(ctrl)
	mockHandler := mock_speech.NewMockRecognitionHandler(ctrl)

	mockVAD.EXPECT().Start(gomock.Any()).Return(nil)
	mockTranscriber.EXPECT().Start(gomock.Any()).Return(tErr)

	r := speech.NewRecognizer(speech.RecognizerConfig{
		VAD:         mockVAD,
		Transcriber: mockTranscriber,
		Handler:     mockHandler,
	})

	err := r.Start(context.Background())
	assert.ErrorIs(t, err, speech.ErrStartingTranscriber)
	assert.ErrorIs(t, err, tErr)
}

func TestRecognizer_NoVAD_FinalTranscript(t *testing.T) {
	ctrl := gomock.NewController(t)

	transcribeCh := make(chan core.Transcript, 2)
	mockTranscriber := mock_speech.NewMockTranscriber(ctrl)
	mockHandler := mock_speech.NewMockRecognitionHandler(ctrl)

	mockTranscriber.EXPECT().Start(gomock.Any()).Return(nil)
	mockTranscriber.EXPECT().Transcribe().Return(transcribeCh)
	mockTranscriber.EXPECT().Stop(gomock.Any()).Return(nil)

	handlerDone := make(chan struct{})
	mockHandler.EXPECT().OnInterim()
	mockHandler.EXPECT().OnSpeechEnd()
	mockHandler.EXPECT().OnSpeechRecognized(gomock.Any()).Do(func(transcripts []core.Transcript) {
		assert.Len(t, transcripts, 1)
		assert.Equal(t, "hello world", transcripts[0].Text)
		close(handlerDone)
	})

	r := speech.NewRecognizer(speech.RecognizerConfig{
		Transcriber: mockTranscriber,
		Handler:     mockHandler,
	})

	require.NoError(t, r.Start(context.Background()))

	transcribeCh <- core.Transcript{Text: "hello", IsFinal: false}
	transcribeCh <- core.Transcript{Text: "hello world", IsFinal: true}

	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for OnSpeechRecognized")
	}

	require.NoError(t, r.Stop(context.Background()))
}

func TestRecognizer_WithVAD_BuffersUntilSpeechEnd(t *testing.T) {
	ctrl := gomock.NewController(t)

	vadCh := make(chan speech.VADEvent, 4)
	transcribeCh := make(chan core.Transcript, 4)
	mockVAD := mock_speech.NewMockVAD(ctrl)
	mockTranscriber := mock_speech.NewMockTranscriber(ctrl)
	mockHandler := mock_speech.NewMockRecognitionHandler(ctrl)

	mockVAD.EXPECT().Start(gomock.Any()).Return(nil)
	mockVAD.EXPECT().Event().Return(vadCh)
	mockVAD.EXPECT().Stop(gomock.Any()).Return(nil)
	mockTranscriber.EXPECT().Start(gomock.Any()).Return(nil)
	mockTranscriber.EXPECT().Transcribe().Return(transcribeCh)
	mockTranscriber.EXPECT().Stop(gomock.Any()).Return(nil)

	handlerDone := make(chan struct{})
	mockHandler.EXPECT().OnSpeechStart()
	mockHandler.EXPECT().OnInterim()
	mockHandler.EXPECT().OnSpeechEnd()
	mockHandler.EXPECT().OnSpeechRecognized(gomock.Any()).Do(func(transcripts []core.Transcript) {
		assert.Len(t, transcripts, 1)
		assert.Equal(t, "hello", transcripts[0].Text)
		close(handlerDone)
	})

	r := speech.NewRecognizer(speech.RecognizerConfig{
		VAD:         mockVAD,
		Transcriber: mockTranscriber,
		Handler:     mockHandler,
	})

	require.NoError(t, r.Start(context.Background()))

	vadCh <- speech.VADEventSpeechStart
	time.Sleep(10 * time.Millisecond)
	transcribeCh <- core.Transcript{Text: "hel", IsFinal: false}
	time.Sleep(10 * time.Millisecond)
	transcribeCh <- core.Transcript{Text: "hello", IsFinal: true}
	time.Sleep(10 * time.Millisecond)
	vadCh <- speech.VADEventSpeechEnd

	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for OnSpeechRecognized")
	}

	require.NoError(t, r.Stop(context.Background()))
}

func TestRecognizer_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)

	vadCh := make(chan speech.VADEvent)
	transcribeCh := make(chan core.Transcript)
	mockVAD := mock_speech.NewMockVAD(ctrl)
	mockTranscriber := mock_speech.NewMockTranscriber(ctrl)
	mockHandler := mock_speech.NewMockRecognitionHandler(ctrl)

	mockVAD.EXPECT().Start(gomock.Any()).Return(nil)
	mockVAD.EXPECT().Event().Return(vadCh)
	mockVAD.EXPECT().Stop(gomock.Any()).Return(nil)
	mockTranscriber.EXPECT().Start(gomock.Any()).Return(nil)
	mockTranscriber.EXPECT().Transcribe().Return(transcribeCh)
	mockTranscriber.EXPECT().Stop(gomock.Any()).Return(nil)

	r := speech.NewRecognizer(speech.RecognizerConfig{
		VAD:         mockVAD,
		Transcriber: mockTranscriber,
		Handler:     mockHandler,
	})

	require.NoError(t, r.Start(context.Background()))
	assert.NoError(t, r.Stop(context.Background()))
}
