package transport

import (
	"errors"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
)

type WebRTCAPIFactory interface {
	Create() (*webrtc.API, error)
}

type DefaultWebRTCAPIFactory struct {
	options *webRTCAPIOptions
}

func NewDefaultWebRTCAPIFactory(opts ...WebRTCAPIOption) *DefaultWebRTCAPIFactory {
	options := buildWebRTCAPIOptions(opts...)

	return &DefaultWebRTCAPIFactory{
		options: options,
	}
}

func (f *DefaultWebRTCAPIFactory) Create() (*webrtc.API, error) {
	mediaEngine := &webrtc.MediaEngine{}

	err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   OpusClockRate,
			Channels:    OpusChannels,
			SDPFmtpLine: OpusSDPFmtpLine,
		},
		PayloadType: OpusPayloadType,
	}, webrtc.RTPCodecTypeAudio)

	if err != nil {
		return nil, errors.Join(ErrRegisterOpusCodec, err)
	}

	interceptorRegistry := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(mediaEngine, interceptorRegistry); err != nil {
		return nil, errors.Join(ErrRegisterInterceptors, err)
	}

	settingEngine := webrtc.SettingEngine{}
	if f.options.portRangeMin > 0 && f.options.portRangeMax > 0 {
		if err := settingEngine.SetEphemeralUDPPortRange(f.options.portRangeMin, f.options.portRangeMax); err != nil {
			return nil, errors.Join(ErrSetPortRange, err)
		}
	}

	if len(f.options.publicIPs) > 0 {
		settingEngine.SetNAT1To1IPs(f.options.publicIPs, webrtc.ICECandidateTypeHost)
	}

	if f.options.iceLite {
		settingEngine.SetLite(true)
	}

	if f.options.interfaceFilter != nil {
		settingEngine.SetInterfaceFilter(f.options.interfaceFilter)
	}

	if f.options.ipFilter != nil {
		settingEngine.SetIPFilter(f.options.ipFilter)
	}

	if len(f.options.candidateTypes) > 0 {
		settingEngine.SetNetworkTypes(f.options.candidateTypes)
	}

	return webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(settingEngine),
		webrtc.WithInterceptorRegistry(interceptorRegistry),
	), nil
}
