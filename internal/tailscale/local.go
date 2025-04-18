package tailscale

import (
	"context"
	"errors"
	"net/netip"
	"strings"

	"tailscale.com/client/tailscale"
)

// Local connects to the local Tailscale instance
type Local struct {
	inner tailscale.LocalClient
}

// NewLocal connects to the local Tailscale socket
func NewLocal() *Local {
	return &Local{
		inner: tailscale.LocalClient{},
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
	status, err := c.inner.StatusWithoutPeers(ctx)
	if err != nil {
		return nil, err
	}

	if status.CurrentTailnet == nil || status.Self == nil {
		return nil, errors.New("current node is uninitialized")
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
