package launcher

import (
	"net/netip"
)

// local notifies a channel of a new verifier launch request
type local struct {
	bus chan<- Request
}

var _ Backend = (*local)(nil)

// NewLocal creates a launcher sending requests via channel to another component
func NewLocal(bus chan<- Request) Backend {
	return &local{bus}
}

func (l *local) Launch(id string, addresses []netip.AddrPort) error {
	l.bus <- Request{
		ID:        id,
		Addresses: addresses,
	}

	return nil
}

// Request contains the information to start a new verification process for the local backend
type Request struct {
	// ID contains the flow identifier the request corresponds to
	ID string
	// Addresses contains the IP address-port pairs to check
	Addresses []netip.AddrPort
}
