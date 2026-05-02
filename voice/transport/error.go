package transport

import "errors"

var (
	ErrRegisterOpusCodec     = errors.New("failed to register Opus codec")
	ErrRegisterInterceptors  = errors.New("failed to register interceptors")
	ErrSetPortRange          = errors.New("failed to set port range")
	ErrCreateWebRTCAPI       = errors.New("failed to create WebRTC API")
	ErrCreatePeerConnection  = errors.New("failed to create PeerConnection")
	ErrSessionClosed         = errors.New("session closed")
	ErrCreateOpusEncoder     = errors.New("failed to create Opus encoder")
	ErrOpusEncode            = errors.New("failed to encode Opus frame")
	ErrAddTrack              = errors.New("failed to add local track")
	ErrSetRemoteDescription  = errors.New("failed to set remote description")
	ErrCreateAnswer          = errors.New("failed to create answer")
	ErrSetLocalDescription   = errors.New("failed to set local description")
	ErrICEGatheringCancelled = errors.New("ICE gathering cancelled")
	ErrDataChannelNotOpen    = errors.New("data channel not open")
	ErrCreateDataChannel     = errors.New("failed to create data channel")
)
