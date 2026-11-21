package tailscale

import (
	"context"
	"crypto/tls"
	"net"
)

// Connection represents an active connection to Tailscale
type Connection interface {
	// Healthy returns whether the node is healthy
	Healthy(ctx context.Context) bool
	// ListenPrivate creates a TCP listener for the requested address
	ListenPrivate(ctx context.Context, address string) (net.Listener, error)
	// ListenFunnel creates a listener exposed to the internet via Tailscale Funnel on port 443
	ListenFunnel(ctx context.Context) (net.Listener, error)
	// DNSName returns the node's MagicDNS name
	DNSName(ctx context.Context) (string, error)
	// GetCertificate fetches a TLS certificate for the TLS client hello
	GetCertificate(hi *tls.ClientHelloInfo) (*tls.Certificate, error)
	// WhoIs looks up the owner of the given address
	//
	// Returns (nil, nil) when peer could not be found
	WhoIs(ctx context.Context, address string) (*Node, error)
}

// Node is a device on the tailnet
type Node struct {
	Hostname string
	ID       string

	Tailnet Tailnet

	// Only one of Tags or User will be set
	Tags []string
	User *User
}

// TagOwned returns true when the node is owned by tags instead of a user
func (p *Node) TagOwned() bool {
	return len(p.Tags) > 0
}

// Tailnet contains details about a particular tailnet
type Tailnet struct {
	// ID is the stable ID for the tailnet. May not be present depending on client version
	ID string
	// Name of the tailnet
	Name string
	// DNS is the Magic DNS suffix (e.g. something-cool.ts.net), if it is enabled
	DNS string
}

type User struct {
	ID     int64
	Login  string
	Groups []string
}
