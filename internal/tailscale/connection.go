package tailscale

import (
	"context"
	"errors"
	"net"
	"slices"
	"strings"

	"tailscale.com/client/local"
	"tailscale.com/client/tailscale/apitype"
	"tailscale.com/ipn/ipnstate"
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

func nodeFromWhoIsResponse(status *ipnstate.Status, res *apitype.WhoIsResponse, err error) (*Node, error) {
	if errors.Is(err, local.ErrPeerNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		var user *User
		if !res.Node.IsTagged() {
			slices.Sort(res.UserProfile.Groups)
			user = &User{
				ID:     int64(res.UserProfile.ID),
				Login:  res.UserProfile.LoginName,
				Groups: res.UserProfile.Groups,
			}
		}

		dns := status.CurrentTailnet.MagicDNSSuffix
		if !status.CurrentTailnet.MagicDNSEnabled {
			dns = ""
		}

		slices.Sort(res.Node.Tags)
		return &Node{
			Tailnet: Tailnet{
				ID:   string(status.CurrentTailnet.StableID),
				Name: status.CurrentTailnet.Name,
				DNS:  dns,
			},
			Hostname: res.Node.ComputedName,
			ID:       string(res.Node.StableID),
			Tags:     res.Node.Tags,
			User:     user,
		}, nil
	}
}
