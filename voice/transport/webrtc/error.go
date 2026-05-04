package webrtc

import (
	"errors"
	"live-agent-go/voice/transport"
)

var (
	ErrRegisterOpusCodec     = errors.New("failed to register Opus codec")
	ErrRegisterInterceptors  = errors.New("failed to register interceptors")
	ErrSetPortRange          = errors.New("failed to set port range")
	ErrCreateAPI             = errors.New("failed to create API")
	ErrCreatePeerConnection  = errors.New("failed to create PeerConnection")
	ErrCreateOpusEncoder     = errors.New("failed to create Opus encoder")
	ErrOpusEncode            = errors.New("failed to encode Opus frame")
	ErrAddTrack              = errors.New("failed to add local track")
	ErrSetRemoteDescription  = errors.New("failed to set remote description")
	ErrCreateAnswer          = errors.New("failed to create answer")
	ErrSetLocalDescription   = errors.New("failed to set local description")
	ErrICEGatheringCancelled = errors.New("ICE gathering cancelled")
	ErrSessionClosed         = transport.ErrSessionClosed
	ErrDataChannelNotOpen    = transport.ErrMessageNotReady
	ErrUnsupportedFrameType  = errors.New("unsupported audio frame type")
)
