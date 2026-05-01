package core

type AudioFrame struct {
	Data       []int16
	SampleRate int32
	Channels   int8
}

type Transcript struct {
	Text    string
	IsFinal bool
}

type Token struct {
	MessageID string
	Text      string
}
