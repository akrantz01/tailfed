package refresher

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/akrantz01/tailfed/internal/api"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/sirupsen/logrus"
)

// Refresher holds the state for periodically refreshing the identity token
type Refresher struct {
	api    *api.Client
	ts     *tailscale.Client
	logger logrus.FieldLogger

	inFlight map[string]inFlight
}

type inFlight struct {
	id            string
	signingSecret string

	listeners []net.Listener
}

// New creates a new Refresher
func New(api *api.Client, ts *tailscale.Client) *Refresher {
	return &Refresher{
		api:    api,
		ts:     ts,
		logger: logrus.WithField("component", "refresher"),

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
		return err
	}

	r.logger.WithField("flow", res.ID).Debug("new flow successfully started")
	r.inFlight[res.ID] = inFlight{
		id:            res.ID,
		signingSecret: res.SigningSecret,
		listeners:     listeners,
	}

	// TODO: start HTTP listener(s)

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

// ShutdownInFlight shuts down all the in-flight refresh flows
func (r *Refresher) ShutdownInFlight() {
	r.logger.Debug("shutting down in-flight flows...")

	for _, flow := range r.inFlight {
		r.releaseListeners(flow.listeners)
	}

	r.logger.Debug("successfully shutdown in-flight requests")
}
