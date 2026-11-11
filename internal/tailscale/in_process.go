package tailscale

import (
	"context"
	"net"
	"os"

	"github.com/sirupsen/logrus"
	"tailscale.com/client/local"
	"tailscale.com/tsnet"
)

// InProcess uses tsnet to connect to Tailscale for the current process only
type InProcess struct {
	logger logrus.FieldLogger
	ts     *tsnet.Server
	client *local.Client
}

var _ Connection = (*InProcess)(nil)

// NewInProcess configures a new in-process connection
func NewInProcess(ctx context.Context, dir string, controlUrl string, tags []string, credentials CredentialProvider) (Connection, error) {
	logger := logrus.
		WithField("component", "tailscale").
		WithField("tailscale.connection", "in-process")

	hostname, err := os.Hostname()
	if err != nil {
		logger.WithError(err).Warn("failed to determine hostname")
		hostname = "tailfed-server"
	}

	ts := &tsnet.Server{
		Dir:           dir,
		Hostname:      hostname,
		UserLogf:      logger.Infof,
		Logf:          logger.Tracef,
		AdvertiseTags: tags,
	}
	credentials.apply(ts)

	logger.
		WithField("tailscale.control", controlUrl).
		WithField("tailscale.tags", tags).
		Debug("connecting to tailscale...")
	status, err := ts.Up(ctx)
	if err != nil {
		return nil, err
	}
	logger.WithField("tailscale.tailnet", status.CurrentTailnet.Name).Info("connected to tailscale")

	client, err := ts.LocalClient()
	if err != nil {
		return nil, err
	}

	return &InProcess{
		logger: logger,
		ts:     ts,
		client: client,
	}, nil
}

func (p *InProcess) Healthy(ctx context.Context) bool {
	status, err := p.client.StatusWithoutPeers(ctx)
	if err != nil {
		p.logger.WithError(err).Error("failed to get node status")
		return false
	}

	return len(status.Health) == 0
}

func (p *InProcess) Listen(_ context.Context, address, service string) (net.Listener, error) {
	if len(service) > 0 {
		return p.ts.ListenService(service, tsnet.ServiceModeHTTP{
			Port:  443,
			HTTPS: true,
		})
	} else {
		return p.ts.Listen("tcp", address)
	}
}

// CredentialProvider provides credentials for authentication with Tailscale
type CredentialProvider interface {
	apply(*tsnet.Server)
}

type authKeyProvider struct {
	key string
}

var _ CredentialProvider = (*authKeyProvider)(nil)

func (a *authKeyProvider) apply(ts *tsnet.Server) {
	ts.AuthKey = a.key
}

// AuthKey authenticates to Tailscale using a static authkey
func AuthKey(key string) CredentialProvider {
	return &authKeyProvider{key}
}

type oauthProvider struct {
	id     string
	secret string
}

var _ CredentialProvider = (*oauthProvider)(nil)

func (p *oauthProvider) apply(ts *tsnet.Server) {
	ts.ClientID = p.id
	ts.ClientSecret = p.secret
}

// OAuth generates authkey(s) via an OAuth exchange
func OAuth(clientId, clientSecret string) CredentialProvider {
	return &oauthProvider{clientId, clientSecret}
}

type workloadProvider struct {
	id       string
	token    string
	audience string
}

var _ CredentialProvider = (*workloadProvider)(nil)

func (p *workloadProvider) apply(ts *tsnet.Server) {
	ts.ClientID = p.id
	ts.IDToken = p.token
	ts.Audience = p.audience
}

// WorkloadIdentityFederation generates authkey(s) using workload identity federation
//
// An existing ID token can be passed which will be used directly. For workloads running in a
// certain set of providers, the ID token can be retrieved automatically for the specified
// audience. The supported providers can be found here: https://tailscale.com/docs/features/workload-identity-federation#supported-environments
func WorkloadIdentityFederation(clientId string, idToken string, audience string) CredentialProvider {
	return &workloadProvider{clientId, idToken, audience}
}
