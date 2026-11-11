package tailscale

import (
	"context"
	"net"

	"tailscale.com/tailcfg"
)

// Connection represents an active connection to Tailscale
type Connection interface {
	// Healthy returns whether the node is healthy
	Healthy(ctx context.Context) bool
	// Listen creates a TCP listener for the request address, binding a service to it when service is nonempty
	Listen(ctx context.Context, address string, service tailcfg.ServiceName) (net.Listener, error)
}
