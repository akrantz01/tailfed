package tailscale

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"tailscale.com/client/local"
	"tailscale.com/ipn"
)

var (
	ErrUninitialized = errors.New("current node is uninitialized")
)

// serveConfigAttempts bounds retries when the serve config is modified concurrently
const serveConfigAttempts = 3

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

func (l *Local) Healthy(ctx context.Context) bool {
	status, err := l.client.StatusWithoutPeers(ctx)
	if err != nil {
		l.logger.WithError(err).Error("failed to get node status")
		return false
	}

	return len(status.Health) == 0
}

func (l *Local) ListenPrivate(_ context.Context, address string) (net.Listener, error) {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to %s: %w", address, err)
	}

	return ln, nil
}

func (l *Local) ListenFunnel(ctx context.Context) (net.Listener, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to bind funnel backend: %w", err)
	}

	fl, err := l.bindFunnel(ctx, ln)
	if err != nil {
		_ = ln.Close()
		return nil, err
	}

	return fl, nil
}

func (l *Local) DNSName(ctx context.Context) (string, error) {
	return dnsName(ctx, &l.client)
}

func (l *Local) WhoIs(ctx context.Context, address string) (*Node, error) {
	status, err := l.client.StatusWithoutPeers(ctx)
	if err != nil {
		return nil, err
	}

	res, err := l.client.WhoIs(ctx, address)
	return nodeFromWhoIsResponse(status, res, err)
}

// bindFunnel serves the listener on port 443 of the node and exposes it via funnel
func (l *Local) bindFunnel(ctx context.Context, ln net.Listener) (*FunnelListener, error) {
	host, err := l.DNSName(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get node status: %w", err)
	}

	proxy := fmt.Sprintf("http://%s", ln.Addr())
	err = l.editServeConfig(ctx, func(sc *ipn.ServeConfig) {
		sc.SetWebHandler(&ipn.HTTPHandler{Proxy: proxy}, host, 443, "/", true, "")
		sc.SetFunnel(host, 443, true)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to funnel %q: %w", host, err)
	}

	l.logger.WithField("tailscale.funnel.host", host).Info("serving tailscale funnel")
	return &FunnelListener{Listener: ln, l: l, host: host}, nil
}

func (l *Local) unfunnel(ctx context.Context, host string) error {
	return l.editServeConfig(ctx, func(sc *ipn.ServeConfig) {
		sc.RemoveWebHandler(host, 443, []string{"/"}, true)
	})
}

// editServeConfig applies edit to the current serve config, retrying if it was modified concurrently
func (l *Local) editServeConfig(ctx context.Context, edit func(*ipn.ServeConfig)) error {
	for attempt := 1; ; attempt++ {
		sc, err := l.client.GetServeConfig(ctx)
		if err != nil {
			return err
		}

		edit(sc)

		err = l.client.SetServeConfig(ctx, sc)
		if err == nil || !local.IsPreconditionsFailedError(err) || attempt == serveConfigAttempts {
			return err
		}

		l.logger.WithField("attempt", attempt).Debug("serve config changed concurrently, retrying")
	}
}

// FunnelListener is a listener whose Close also removes its funnel
type FunnelListener struct {
	net.Listener

	l    *Local
	host string

	closeOnce sync.Once
	closeErr  error
}

func (fl *FunnelListener) Close() error {
	fl.closeOnce.Do(func() {
		// Close has no context
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		fl.closeErr = errors.Join(
			fl.l.unfunnel(ctx, fl.host),
			fl.Listener.Close(),
		)
	})
	return fl.closeErr
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
