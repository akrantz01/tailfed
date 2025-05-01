package launcher

import (
	"net/netip"

	"github.com/sirupsen/logrus"
)

// local notifies a channel of a new verifier launch request
type local struct {
	logger logrus.FieldLogger
	bus    chan<- Request
}

var _ Backend = (*local)(nil)

// NewLocal creates a launcher sending requests via channel to another component
func NewLocal(logger logrus.FieldLogger, bus chan<- Request) Backend {
	logger.Info("created new local launcher")
	return &local{logger, bus}
}

func (l *local) Launch(id string, addresses []netip.AddrPort) error {
	l.logger.WithField("id", id).Debug("sending launch request across channel...")
	l.bus <- Request{
		ID:        id,
		Addresses: addresses,
	}

	l.logger.Debug("launch request sent")
	return nil
}
