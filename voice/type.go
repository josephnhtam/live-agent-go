package voice

import (
	"live-agent-go/voice/core"
	"live-agent-go/voice/internal/speech"
)

type AudioFrame = core.AudioFrame
type Transcript = core.Transcript
type Token = core.Token

type VADEvent = speech.VADEvent

var (
	VADEventSpeechStart VADEvent = speech.VADEventSpeechStart
	VADEventSpeechEnd   VADEvent = speech.VADEventSpeechEnd
)
