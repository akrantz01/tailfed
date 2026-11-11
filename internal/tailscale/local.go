package tailscale

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"tailscale.com/client/local"
)

var ErrUninitialized = errors.New("current node is uninitialized")

// Local connects to the local Tailscale instance
type Local struct {
	logger logrus.FieldLogger
	client local.Client
}

var _ Connection = (*Local)(nil)

// NewLocal connects to the local Tailscale socket
func NewLocal(ctx context.Context) (*Local, error) {
	l := &Local{
		logger: logrus.WithField("component", "tailscale").WithField("tailscale.connection", "local"),
		client: local.Client{},
	}

	for {
		status, err := l.client.StatusWithoutPeers(ctx)
		if err != nil {
			return nil, err
		}

		if status.BackendState == "Running" {
			l.logger.WithField("tailscale.tailnet", status.CurrentTailnet.Name).Info("connected to tailscale")
			break
		}

		l.logger.WithField("tailscale.state", status.BackendState).Warn("tailscale is not running, retrying in 15s...")
		time.Sleep(15 * time.Second)
	}

	return l, nil
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
func (l *Local) Status(ctx context.Context) (*Status, error) {
	l.logger.Debug("getting local tailscale node info")
	status, err := l.client.StatusWithoutPeers(ctx)
	if err != nil {
		return nil, err
	}

	if status.CurrentTailnet == nil || status.Self == nil {
		l.logger.Debug("client is uninitialized")
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
