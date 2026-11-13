package tailscale

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"tailscale.com/client/local"
	"tailscale.com/client/tailscale/apitype"
	"tailscale.com/tailcfg"
	"tailscale.com/tsnet"
)

// AuthenticationMethod determines how tailfed should authenticate for in-process connections
type AuthenticationMethod string

const (
	// AuthenticationMethodKey uses a static authkey
	AuthenticationMethodKey AuthenticationMethod = "auth-key"
	// AuthenticationMethodOAuth uses the OAuth flow for generating authkey(s)
	AuthenticationMethodOAuth AuthenticationMethod = "oauth"
	// AuthenticationMethodFederation uses workload identity federation for generating authkey(s)
	AuthenticationMethodFederation AuthenticationMethod = "federation"
)

var AuthenticationMethods string = strings.Join([]string{string(AuthenticationMethodKey), string(AuthenticationMethodOAuth), string(AuthenticationMethodFederation)}, ", ")

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

func (p *InProcess) Listen(_ context.Context, address string, service tailcfg.ServiceName) (net.Listener, error) {
	if len(service) > 0 {
		return p.ts.ListenService(string(service), tsnet.ServiceModeHTTP{
			Port:  443,
			HTTPS: true,
		})
	} else {
		return p.ts.Listen("tcp", address)
	}
}

func (p *InProcess) WhoIs(ctx context.Context, address string) (*apitype.WhoIsResponse, error) {
	res, err := p.client.WhoIs(ctx, address)
	if errors.Is(err, local.ErrPeerNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		return res, nil
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
