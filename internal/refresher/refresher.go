package refresher

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/akrantz01/tailfed/internal/api"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/sirupsen/logrus"
)

// Refresher holds the state for periodically refreshing the identity token
type Refresher struct {
	api    *api.Client
	ts     *tailscale.Local
	logger logrus.FieldLogger

	path string

	inFlight map[string]inFlight
}

type inFlight struct {
	listeners []net.Listener
	servers   []*http.Server
}

// New creates a new Refresher
func New(api *api.Client, ts *tailscale.Local, path string) *Refresher {
	return &Refresher{
		api:    api,
		ts:     ts,
		logger: logrus.WithField("component", "refresher"),

		path: path,

		inFlight: make(map[string]inFlight),
	}
}

// Job performs a single run of the refresh flow
func (r *Refresher) Job(ctx context.Context) error {
	status, err := r.ts.Status(ctx)
	if err != nil {
		return err
	}

	if !status.Ready {
		return errors.New("node is not ready")
	} else if !status.Healthy {
		r.logger.Warn("node is unhealthy")
	}

	listeners, addresses, err := r.bindListeners(status.IPs)
	if err != nil {
		return fmt.Errorf("failed to bind listeners: %w", err)
	}

	res, err := r.api.Start(ctx, status.ID, addresses)
	if err != nil {
		r.releaseListeners(listeners)
		return err
	}

	servers := make([]*http.Server, 0, len(listeners))
	for _, lis := range listeners {
		servers = append(servers, r.launchServer(res.ID, res.SigningSecret, lis))
	}

	r.logger.WithField("flow", res.ID).Debug("new flow successfully started")
	r.inFlight[res.ID] = inFlight{
		listeners: listeners,
		servers:   servers,
	}

	return nil
}

func (r *Refresher) bindListeners(ips []netip.Addr) ([]net.Listener, []string, error) {
	listeners := make([]net.Listener, 0, len(ips))
	addresses := make([]string, 0, len(ips))

	defer func() {
		if len(listeners) != len(ips) {
			r.logger.Debug("all listeners not bound successfully, releasing...")
			r.releaseListeners(listeners)
		}
	}()

	for _, ip := range ips {
		lis, err := net.Listen("tcp", netip.AddrPortFrom(ip, 0).String())
		if err != nil {
			return nil, nil, err
		}

		addr := lis.Addr().String()
		r.logger.WithField("address", addr).Debug("bound listener")

		listeners = append(listeners, lis)
		addresses = append(addresses, addr)
	}
	r.logger.WithField("count", len(listeners)).Debug("successfully bound listeners")

	return listeners, addresses, nil
}

func (r *Refresher) releaseListeners(listeners []net.Listener) {
	for _, lis := range listeners {
		logger := r.logger.WithField("address", lis.Addr().String())

		if err := lis.Close(); err != nil {
			logger.WithError(err).Error("failed to release listener")
		} else {
			logger.Debug("shutdown listener")
		}
	}
}

func (r *Refresher) stopServers(servers []*http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, srv := range servers {
		if err := srv.Shutdown(ctx); err != nil {
			r.logger.WithError(err).Error("failed to shutdown server")
		}
	}
}

// ShutdownInFlight shuts down all the in-flight refresh flows
func (r *Refresher) ShutdownInFlight() {
	r.logger.Debug("shutting down in-flight flows...")

	for _, flow := range r.inFlight {
		r.stopServers(flow.servers)
	}

	r.logger.Debug("successfully shutdown in-flight requests")
}
