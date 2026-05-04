package core

import "context"

type AudioFrame interface {
	SampleRate() int32
	Channels() int8
	Context() context.Context
	SetContext(context.Context)
}

type PCMFrame struct {
	PCMData      []int16
	SampleRateHz int32
	NumChannels  int8
	Ctx          context.Context
}

func (f *PCMFrame) SampleRate() int32              { return f.SampleRateHz }
func (f *PCMFrame) Channels() int8                 { return f.NumChannels }
func (f *PCMFrame) Context() context.Context       { return f.Ctx }
func (f *PCMFrame) SetContext(ctx context.Context)  { f.Ctx = ctx }

type OpusFrame struct {
	OpusData     []byte
	SampleRateHz int32
	NumChannels  int8
	Ctx          context.Context
}

func (f *OpusFrame) SampleRate() int32              { return f.SampleRateHz }
func (f *OpusFrame) Channels() int8                 { return f.NumChannels }
func (f *OpusFrame) Context() context.Context       { return f.Ctx }
func (f *OpusFrame) SetContext(ctx context.Context)  { f.Ctx = ctx }

type Transcript struct {
	Text    string
	IsFinal bool
}

type Token struct {
	MessageID string
	Text      string
}
