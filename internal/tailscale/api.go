package tailscale

import (
	"context"
	"net/netip"

	"tailscale.com/client/tailscale/v2"
)

// API connects to the Tailscale control plane
type API struct {
	inner *tailscale.Client
}

// NewAPI creates a new control plane API client
func NewAPI(tailnet string, auth Authentication) *API {
	client := &tailscale.Client{Tailnet: tailnet}
	auth.apply(client)

	return &API{client}
}

// Tailnet retrieves the name of the connected tailnet
func (a *API) Tailnet() string {
	return a.inner.Tailnet
}

type NodeInfo struct {
	// The ID used by the control plane
	ID string
	// The IP addresses in the network
	Addresses []netip.Addr
	// The unique public key for the machine
	Key string
	// The DNS name of the node within the network
	DNSName string
	// The machine's operating system
	OS string
}

// NodeInfo retrieves details about a particular node by its ID
func (a *API) NodeInfo(ctx context.Context, id string) (*NodeInfo, error) {
	node, err := a.inner.Devices().Get(ctx, id)
	if err != nil {
		if tailscale.IsNotFound(err) {
			err = nil
		}

		return nil, err
	}

	addresses := make([]netip.Addr, 0, len(node.Addresses))
	for _, raw := range node.Addresses {
		addresses = append(addresses, netip.MustParseAddr(raw))
	}

	return &NodeInfo{
		ID:        node.NodeID,
		Addresses: addresses,
		Key:       node.NodeKey,
		DNSName:   node.Name,
		OS:        node.OS,
	}, nil
}
