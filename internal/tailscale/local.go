package tailscale

import (
	"context"
	"errors"
	"net/netip"
	"strings"

	"github.com/sirupsen/logrus"
	"tailscale.com/client/local"
)

var ErrUninitialized = errors.New("current node is uninitialized")

// Local connects to the local Tailscale instance
type Local struct {
	logger logrus.FieldLogger
	inner  local.Client
}

// NewLocal connects to the local Tailscale socket
func NewLocal(logger logrus.FieldLogger) *Local {
	return &Local{
		logger: logger,
		inner:  local.Client{},
	}
}

// Status provides information about the node
type Status struct {
	// Whether the node is ready (authenticated + connected)
	Ready bool
	// Whether there are any health issues raised
	Healthy bool

	// The name of the tailnet
	Tailnet string
	// The IP addresses in the network
	IPs []netip.Addr

	// The ID of the node according to the Tailscale API
	ID string
	// The unique public key for the machine
	PublicKey string
	// The DNS name of the node within the network
	DNSName string
	// The operating system being used
	OS string
}

// Status retrieves information about the current node
func (c *Local) Status(ctx context.Context) (*Status, error) {
	c.logger.Debug("getting local tailscale node info")
	status, err := c.inner.StatusWithoutPeers(ctx)
	if err != nil {
		return nil, err
	}

	if status.CurrentTailnet == nil || status.Self == nil {
		c.logger.Debug("client is uninitialized")
		return nil, ErrUninitialized
	}

	return &Status{
		Ready:     status.BackendState == "Running",
		Healthy:   len(status.Health) == 0,
		Tailnet:   status.CurrentTailnet.Name,
		IPs:       status.TailscaleIPs,
		ID:        string(status.Self.ID),
		PublicKey: status.Self.PublicKey.String(),
		DNSName:   strings.TrimSuffix(status.Self.DNSName, "."),
		OS:        status.Self.OS,
	}, nil
}
