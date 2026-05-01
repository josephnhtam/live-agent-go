package speech

type VADEvent string

const (
	VADEventSpeechStart VADEvent = "speech_start"
	VADEventSpeechEnd   VADEvent = "speech_end"
)
