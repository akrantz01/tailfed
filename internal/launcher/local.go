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
