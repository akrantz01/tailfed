package launcher

import (
	"net/netip"

	"github.com/akrantz01/tailfed/internal/types"
)

// local notifies a channel of a new verifier launch request
type local struct {
	bus chan<- types.VerifyRequest
}

var _ Backend = (*local)(nil)

// NewLocal creates a launcher sending requests via channel to another component
func NewLocal(bus chan<- types.VerifyRequest) Backend {
	return &local{bus}
}

func (l *local) Launch(id string, addresses []netip.AddrPort) error {
	l.bus <- types.VerifyRequest{
		ID:        id,
		Addresses: addresses,
	}

	return nil
}
