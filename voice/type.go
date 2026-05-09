package voice

import (
	"github.com/josephnhtam/live-agent-go/voice/internal/core"
	"github.com/josephnhtam/live-agent-go/voice/internal/dialog"
	"github.com/josephnhtam/live-agent-go/voice/internal/speech"
	"io"
)

type AudioFrame = core.AudioFrame
type PCMFrame = core.PCMFrame
type OpusFrame = core.OpusFrame
type Transcript = core.Transcript
type Token = core.Token
type Prompt = core.Prompt

type VADEvent = speech.VADEvent

var (
	VADEventSpeechStart VADEvent = speech.VADEventSpeechStart
	VADEventSpeechEnd   VADEvent = speech.VADEventSpeechEnd
)

type Wave = dialog.Wave

var (
	LoadWave                         func(data []byte) (*Wave, error)                       = dialog.LoadWave
	LoadWaveFromReader               func(r io.ReadSeeker) (*Wave, error)                   = dialog.LoadWaveFromReader
	LoadWaveWithSampleRate           func(data []byte, targetRate int32) (*Wave, error)     = dialog.LoadWaveWithSampleRate
	LoadWaveFromReaderWithSampleRate func(r io.ReadSeeker, targetRate int32) (*Wave, error) = dialog.LoadWaveFromReaderWithSampleRate
)

type AudioOptions = dialog.AudioOptions

var NewAudioOptions func() *AudioOptions = dialog.NewAudioOptions
