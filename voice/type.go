package voice

import (
	"github.com/josephnhtam/live-agent-go/voice/internal/core"
	"github.com/josephnhtam/live-agent-go/voice/internal/speech"
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
