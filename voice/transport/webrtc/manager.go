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

	if options == nil {
		options = NewManagerOptions()
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
				_ = session.Close(context.Background())
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

func (m *Manager) CreateOffer(ctx context.Context) (string, voice.Session, AcceptAnswerFunc, error) {
	pc, err := m.api.NewPeerConnection(webrtc.Configuration{
		ICEServers: m.options.iceServers,
	})
	if err != nil {
		return "", nil, nil, errors.Join(ErrCreatePeerConnection, err)
	}

	success := false
	var session *Session
	defer func() {
		if !success {
			if session != nil {
				_ = session.Close(context.Background())
			} else {
				_ = pc.Close()
			}
		}
	}()

	session, err = newSession(pc, m.options)
	if err != nil {
		return "", nil, nil, err
	}

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return "", nil, nil, errors.Join(ErrCreateOffer, err)
	}

	if err := pc.SetLocalDescription(offer); err != nil {
		return "", nil, nil, errors.Join(ErrSetLocalDescription, err)
	}

	select {
	case <-webrtc.GatheringCompletePromise(pc):
	case <-ctx.Done():
		return "", nil, nil, errors.Join(ErrICEGatheringCancelled, ctx.Err())
	}

	acceptAnswer := func(answerSDP string) error {
		answer := webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  answerSDP,
		}
		if err := pc.SetRemoteDescription(answer); err != nil {
			return errors.Join(ErrSetRemoteDescription, err)
		}
		return nil
	}

	success = true
	return pc.LocalDescription().SDP, session, acceptAnswer, nil
}

type AcceptAnswerFunc func(answerSDP string) error
