package tailscale

import (
	"context"
	"net"
	"strings"

	"tailscale.com/client/tailscale/apitype"
	"tailscale.com/tailcfg"
)

// ConnectionId identifies a connection in the configuration
type ConnectionId string

const (
	// ConnectionLocal uses an existing Tailscale daemon running on the host
	ConnectionLocal ConnectionId = "local"
	// ConnectionInProcess connects to Tailscale for the current process only
	ConnectionInProcess ConnectionId = "in-process"
)

func (c ConnectionId) Choices() []string {
	return []string{string(ConnectionLocal), string(ConnectionInProcess)}
}

var ConnectionIds string = strings.Join(ConnectionLocal.Choices(), ", ")

// Connection represents an active connection to Tailscale
type Connection interface {
	// Healthy returns whether the node is healthy
	Healthy(ctx context.Context) bool
	// Listen creates a TCP listener for the request address, binding a service to it when service is nonempty
	Listen(ctx context.Context, address string, service tailcfg.ServiceName) (net.Listener, error)
	// WhoIs looks up the owner of the given address
	//
	// Returns (nil, nil) when peer could not be found
	WhoIs(ctx context.Context, address string) (*apitype.WhoIsResponse, error)
}
