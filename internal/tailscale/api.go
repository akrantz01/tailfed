package tailscale

import (
	"context"
	"fmt"
	"net/netip"
	"net/url"

	"tailscale.com/client/tailscale/v2"
)

// API connects to the Tailscale control plane
type API struct {
	inner *tailscale.Client
}

// NewAPI creates a new control plane API client
func NewAPI(baseUrl, tailnet string, auth Authentication) (*API, error) {
	base, err := url.Parse(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid base url %q: %w", baseUrl, err)
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return nil, fmt.Errorf("invalid base url scheme %q", base.Scheme)
	}

	client := &tailscale.Client{
		BaseURL: base,
		Tailnet: tailnet,
	}
	auth.apply(client)

	return &API{client}, nil
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
	// The hostname of the node
	Hostname string
	// The name of the Tailnet
	Tailnet string
	// The machine's operating system
	OS string
	// All ACL tags that are applied to the machine
	Tags []string
	// Whether the device is authorized to join the tailnet
	Authorized bool
	// Whether the device is shared in to the tailnet
	External bool
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
		ID:         node.NodeID,
		Addresses:  addresses,
		Key:        node.NodeKey,
		DNSName:    node.Name,
		Hostname:   node.Hostname,
		Tailnet:    a.inner.Tailnet,
		OS:         node.OS,
		Tags:       node.Tags,
		Authorized: node.Authorized,
		External:   node.IsExternal,
	}, nil
}
