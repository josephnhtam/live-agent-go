package transport

import (
	"context"
	"errors"
	"log/slog"

	"github.com/pion/webrtc/v3"
)

type WebRTCManagerConfig struct {
	ICEServers            []webrtc.ICEServer
	PCMChannelBufferSize  int
	TextChannelBufferSize int
	Logger                *slog.Logger
}

type WebRTCManager struct {
	api    *webrtc.API
	config WebRTCManagerConfig
}

func NewWebRTCManager(apiFactory WebRTCAPIFactory, config WebRTCManagerConfig) (*WebRTCManager, error) {
	if config.PCMChannelBufferSize <= 0 {
		config.PCMChannelBufferSize = 128
	}

	if config.TextChannelBufferSize <= 0 {
		config.TextChannelBufferSize = 16
	}

	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	api, err := apiFactory.Create()
	if err != nil {
		return nil, errors.Join(ErrCreateWebRTCAPI, err)
	}

	return &WebRTCManager{
		api:    api,
		config: config,
	}, nil
}

func (m *WebRTCManager) AcceptOffer(ctx context.Context, offerSDP string) (string, Session, error) {
	pc, err := m.api.NewPeerConnection(webrtc.Configuration{
		ICEServers: m.config.ICEServers,
	})
	if err != nil {
		return "", nil, errors.Join(ErrCreatePeerConnection, err)
	}

	success := false
	var session *WebRTCSession
	defer func() {
		if !success {
			if session != nil {
				_ = session.Close()
			} else {
				_ = pc.Close()
			}
		}
	}()

	outTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   OpusClockRate,
			Channels:    OpusChannels,
			SDPFmtpLine: OpusSDPFmtpLine,
		},
		"audio",
		"voice-agent",
	)
	if err != nil {
		return "", nil, errors.Join(ErrAddTrack, err)
	}

	if _, err := pc.AddTrack(outTrack); err != nil {
		return "", nil, errors.Join(ErrAddTrack, err)
	}

	dataChannel, err := pc.CreateDataChannel(TextDataChannelName, nil)
	if err != nil {
		return "", nil, errors.Join(ErrCreateDataChannel, err)
	}

	session, err = newWebRTCSession(pc, outTrack, dataChannel, m.config.PCMChannelBufferSize, m.config.TextChannelBufferSize, m.config.Logger)
	if err != nil {
		return "", nil, err
	}

	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  offerSDP,
	}
	if err := pc.SetRemoteDescription(offer); err != nil {
		return "", nil, errors.Join(ErrSetRemoteDescription, err)
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return "", nil, errors.Join(ErrCreateAnswer, err)
	}

	if err := pc.SetLocalDescription(answer); err != nil {
		return "", nil, errors.Join(ErrSetLocalDescription, err)
	}

	select {
	case <-webrtc.GatheringCompletePromise(pc):
	case <-ctx.Done():
		return "", nil, errors.Join(ErrICEGatheringCancelled, ctx.Err())
	}

	success = true
	return pc.LocalDescription().SDP, session, nil
}
