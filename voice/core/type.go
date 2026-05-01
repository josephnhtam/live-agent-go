package core

type AudioFrame struct {
	Data       []any
	SampleRate int32
	Channels   int8
}

type Transcript struct {
	Text    string
	IsFinal bool
}

type Token struct {
	Text string
}
