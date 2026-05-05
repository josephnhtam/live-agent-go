package webrtc

import (
	"context"
	"errors"
	"github.com/josephnhtam/live-agent-go/voice"
	"github.com/pion/webrtc/v3"
)

type Manager struct {
	api     *webrtc.API
	options *ManagerOptions
}

func NewManager(apiFactory APIFactory, options *ManagerOptions) (*Manager, error) {
	api, err := apiFactory.Create()
	if err != nil {
		return nil, errors.Join(ErrCreateAPI, err)
	}

	return &Manager{
		api:     api,
		options: options,
	}, nil
}

func (m *Manager) AcceptOffer(ctx context.Context, offerSDP string) (string, voice.Session, error) {
	pc, err := m.api.NewPeerConnection(webrtc.Configuration{
		ICEServers: m.options.iceServers,
	})
	if err != nil {
		return "", nil, errors.Join(ErrCreatePeerConnection, err)
	}

	success := false
	var session *Session
	defer func() {
		if !success {
			if session != nil {
				_ = session.Close()
			} else {
				_ = pc.Close()
			}
		}
	}()

	session, err = newSession(pc, m.options)
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
