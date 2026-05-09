package webrtc

import (
	"context"
	"io"
	"log/slog"
	"sync"

	"github.com/josephnhtam/live-agent-go/voice/helper"
	"github.com/josephnhtam/live-agent-go/voice/internal/core"

	"github.com/pion/webrtc/v3"
	"gopkg.in/hraban/opus.v2"
)

type audioReceiver struct {
	audioIn         chan core.AudioFrame
	audioInEncoding AudioInEncoding
	trackWg         sync.WaitGroup
	ctx             context.Context
	logger          *slog.Logger
}

func newAudioReceiver(bufSize int, encoding AudioInEncoding, ctx context.Context, logger *slog.Logger) *audioReceiver {
	return &audioReceiver{
		audioIn:         make(chan core.AudioFrame, bufSize),
		audioInEncoding: encoding,
		ctx:             ctx,
		logger:          logger,
	}
}

func (r *audioReceiver) AudioIn() <-chan core.AudioFrame {
	return r.audioIn
}

func (r *audioReceiver) OnTrack(track *webrtc.TrackRemote) {
	if track.Codec().MimeType != webrtc.MimeTypeOpus {
		return
	}

	r.trackWg.Add(1)
	go r.handleAudioTrack(track)
}

func (r *audioReceiver) handleAudioTrack(track *webrtc.TrackRemote) {
	defer r.trackWg.Done()

	switch r.audioInEncoding {
	case AudioInEncodingOpus:
		r.handleAudioTrackOpus(track)
	default:
		r.handleAudioTrackPCM(track)
	}
}

func (r *audioReceiver) handleAudioTrackPCM(track *webrtc.TrackRemote) {
	decoder, err := opus.NewDecoder(OpusClockRate, PCMDecoderChannels)
	if err != nil {
		r.logger.Error("failed to create Opus decoder", "error", err)
		return
	}

	pcmBuffer := make([]int16, PCMBufferSize)

	for {
		rtpPacket, _, err := track.ReadRTP()
		if err != nil {
			if err != io.EOF {
				r.logger.Error("error reading RTP", "error", err)
			}
			return
		}

		if len(rtpPacket.Payload) == 0 {
			continue
		}

		n, err := decoder.Decode(rtpPacket.Payload, pcmBuffer)
		if err != nil {
			r.logger.Warn("error decoding Opus", "error", err)
			continue
		}

		pcmData := make([]int16, n)
		copy(pcmData, pcmBuffer[:n])

		frame := &core.PCMFrame{
			PCMData:      pcmData,
			SampleRateHz: OpusClockRate,
			NumChannels:  PCMDecoderChannels,
		}

		select {
		case r.audioIn <- frame:
		case <-r.ctx.Done():
			return
		default:
			r.logger.Warn("audio input channel full, dropping audio frame")
		}
	}
}

func (r *audioReceiver) handleAudioTrackOpus(track *webrtc.TrackRemote) {
	for {
		rtpPacket, _, err := track.ReadRTP()
		if err != nil {
			if err != io.EOF {
				r.logger.Error("error reading RTP", "error", err)
			}
			return
		}

		if len(rtpPacket.Payload) == 0 {
			continue
		}

		opusData := make([]byte, len(rtpPacket.Payload))
		copy(opusData, rtpPacket.Payload)

		frame := &core.OpusFrame{
			OpusData:     opusData,
			SampleRateHz: OpusClockRate,
			NumChannels:  OpusChannels,
		}

		select {
		case r.audioIn <- frame:
		case <-r.ctx.Done():
			return
		default:
			r.logger.Warn("audio input channel full, dropping audio frame")
		}
	}
}

func (r *audioReceiver) Wait(ctx context.Context) error {
	return helper.WaitWithCtx(ctx, &r.trackWg)
}

func (r *audioReceiver) Close() {
	close(r.audioIn)
}
