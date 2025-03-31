package refresher

import (
	"context"
	"net"
	"net/http"
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
