package transport

import (
	"net"

	"github.com/pion/webrtc/v3"
)

type webRTCAPIOptions struct {
	portRangeMin    uint16
	portRangeMax    uint16
	publicIPs       []string
	iceLite         bool
	interfaceFilter func(string) bool
	ipFilter        func(net.IP) bool
	candidateTypes  []webrtc.NetworkType
}

type WebRTCAPIOption interface {
	apply(*webRTCAPIOptions)
}

type WebRTCAPIOptionFunc func(*webRTCAPIOptions)

func (f WebRTCAPIOptionFunc) apply(options *webRTCAPIOptions) {
	f(options)
}

func buildWebRTCAPIOptions(opts ...WebRTCAPIOption) *webRTCAPIOptions {
	options := &webRTCAPIOptions{}

	for _, opt := range opts {
		opt.apply(options)
	}

	return options
}

func WithPortRange(portMin, portMax uint16) WebRTCAPIOption {
	return WebRTCAPIOptionFunc(func(options *webRTCAPIOptions) {
		options.portRangeMin = portMin
		options.portRangeMax = portMax
	})
}

func WithPublicIPs(publicIPs []string) WebRTCAPIOption {
	return WebRTCAPIOptionFunc(func(options *webRTCAPIOptions) {
		options.publicIPs = publicIPs
	})
}

func WithICELite() WebRTCAPIOption {
	return WebRTCAPIOptionFunc(func(options *webRTCAPIOptions) {
		options.iceLite = true
	})
}

func WithInterfaceFilter(filter func(string) bool) WebRTCAPIOption {
	return WebRTCAPIOptionFunc(func(options *webRTCAPIOptions) {
		options.interfaceFilter = filter
	})
}

func WithIPFilter(filter func(net.IP) bool) WebRTCAPIOption {
	return WebRTCAPIOptionFunc(func(options *webRTCAPIOptions) {
		options.ipFilter = filter
	})
}

func WithNetworkTypes(candidateTypes []webrtc.NetworkType) WebRTCAPIOption {
	return WebRTCAPIOptionFunc(func(options *webRTCAPIOptions) {
		options.candidateTypes = candidateTypes
	})
}
