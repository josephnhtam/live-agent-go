package transport

import "time"

const (
	OpusClockRate           = 48000
	OpusChannels            = 1
	OpusPayloadType         = 111
	OpusMaxFrameDurationMs  = 60
	OpusSDPFmtpLine         = "minptime=10;useinbandfec=1"
	OpusFrameDuration       = 20 * time.Millisecond
	OpusMaxEncodedFrameSize = 1276

	PCMDecoderChannels = 1
	PCMEncoderChannels = 1
	PCMBufferSize      = OpusClockRate * OpusMaxFrameDurationMs / 1000

	TextDataChannelName = "text"
)
