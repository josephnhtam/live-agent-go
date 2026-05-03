package webrtc

import (
	"net"

	"github.com/pion/webrtc/v3"
)

type APIOptions struct {
	portRangeMin    uint16
	portRangeMax    uint16
	publicIPs       []string
	iceLite         bool
	interfaceFilter func(string) bool
	ipFilter        func(net.IP) bool
	candidateTypes  []webrtc.NetworkType
}

func NewAPIOptions() *APIOptions {
	return &APIOptions{}
}

func (o *APIOptions) WithPortRange(portMin, portMax uint16) *APIOptions {
	o.portRangeMin = portMin
	o.portRangeMax = portMax
	return o
}

func (o *APIOptions) WithPublicIPs(publicIPs []string) *APIOptions {
	o.publicIPs = publicIPs
	return o
}

func (o *APIOptions) WithICELite() *APIOptions {
	o.iceLite = true
	return o
}

func (o *APIOptions) WithInterfaceFilter(filter func(string) bool) *APIOptions {
	o.interfaceFilter = filter
	return o
}

func (o *APIOptions) WithIPFilter(filter func(net.IP) bool) *APIOptions {
	o.ipFilter = filter
	return o
}

func (o *APIOptions) WithNetworkTypes(candidateTypes []webrtc.NetworkType) *APIOptions {
	o.candidateTypes = candidateTypes
	return o
}
