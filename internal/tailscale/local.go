package tailscale

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"tailscale.com/client/local"
	"tailscale.com/ipn"
	"tailscale.com/tailcfg"
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

func (l *Local) Listen(ctx context.Context, address string, service tailcfg.ServiceName) (net.Listener, error) {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to %s: %w", address, err)
	}

	if len(service) == 0 {
		return ln, nil
	}

	sl, err := l.bindService(ctx, ln, service)
	if err != nil {
		_ = ln.Close()
		return nil, err
	}

	return sl, nil
}

func (l *Local) WhoIs(ctx context.Context, address string) (*Node, error) {
	status, err := l.client.StatusWithoutPeers(ctx)
	if err != nil {
		return nil, err
	}

	res, err := l.client.WhoIs(ctx, address)
	return nodeFromWhoIsResponse(status, res, err)
}

// bindService serves and advertises the service, proxying to the listener
func (l *Local) bindService(ctx context.Context, ln net.Listener, service tailcfg.ServiceName) (*ServiceListener, error) {
	status, err := l.client.StatusWithoutPeers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get node status: %w", err)
	}

	port := ln.Addr().(*net.TCPAddr).Port
	proxy := fmt.Sprintf("http://127.0.0.1:%d", port)
	suffix := status.CurrentTailnet.MagicDNSSuffix
	fqdn := service.WithoutPrefix() + "." + suffix

	// Serve before advertising so the service is never routable without a handler
	if err := l.serve(ctx, service, proxy, suffix); err != nil {
		return nil, fmt.Errorf("failed to serve %q: %w", service, err)
	}

	if err := l.setAdvertised(ctx, service, true); err != nil {
		err = fmt.Errorf("failed to advertise service %q: %w", service, err)
		return nil, errors.Join(err, l.unserve(ctx, service, fqdn))
	}

	l.logger.WithField("tailscale.service", service).WithField("tailscale.service.fqdn", fqdn).Info("serving tailscale service")
	return &ServiceListener{Listener: ln, l: l, service: service, fqdn: fqdn}, nil
}

// serve configures an HTTPS handler on port 443 of the service that proxies to the target
func (l *Local) serve(ctx context.Context, service tailcfg.ServiceName, proxy, suffix string) error {
	return l.editServeConfig(ctx, func(sc *ipn.ServeConfig) {
		delete(sc.Services, service)
		sc.SetWebHandler(&ipn.HTTPHandler{Proxy: proxy}, service.String(), 443, "/", true, suffix)
	})
}

// unserve stops advertising the service, then removes its handler
func (l *Local) unserve(ctx context.Context, service tailcfg.ServiceName, fqdn string) error {
	var unadvertiseErr error
	if err := l.setAdvertised(ctx, service, false); err != nil {
		unadvertiseErr = fmt.Errorf("failed to unadvertise service %q: %w", service, err)
	}

	var unserveErr error
	err := l.editServeConfig(ctx, func(sc *ipn.ServeConfig) {
		sc.RemoveServiceWebHandler(service, fqdn, 443, []string{"/"})
	})
	if err != nil {
		unserveErr = fmt.Errorf("failed to stop serving %q: %w", service, err)
	}

	return errors.Join(unadvertiseErr, unserveErr)
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

// setAdvertised adds or removes the service from the node's advertised services
func (l *Local) setAdvertised(ctx context.Context, service tailcfg.ServiceName, advertise bool) error {
	prefs, err := l.client.GetPrefs(ctx)
	if err != nil {
		return err
	}

	name := service.String()
	if slices.Contains(prefs.AdvertiseServices, name) == advertise {
		return nil
	}

	var services []string
	if advertise {
		services = append(prefs.AdvertiseServices, name)
	} else {
		services = slices.DeleteFunc(prefs.AdvertiseServices, func(s string) bool { return s == name })
	}

	_, err = l.client.EditPrefs(ctx, &ipn.MaskedPrefs{
		AdvertiseServicesSet: true,
		Prefs:                ipn.Prefs{AdvertiseServices: services},
	})
	return err
}

// ServiceListener is a listener whose Close also unadvertises and unserves its Tailscale service
type ServiceListener struct {
	net.Listener

	l       *Local
	service tailcfg.ServiceName
	fqdn    string

	closeOnce sync.Once
	closeErr  error
}

func (sl *ServiceListener) Close() error {
	sl.closeOnce.Do(func() {
		// Close has no context
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		sl.closeErr = errors.Join(
			sl.l.unserve(ctx, sl.service, sl.fqdn),
			sl.Listener.Close(),
		)
	})
	return sl.closeErr
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
