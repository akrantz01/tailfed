package tailscale

import (
	"os"

	"github.com/sirupsen/logrus"
	"tailscale.com/tsnet"
)

// InProcess uses tsnet to connect to Tailscale for the current process only
type InProcess struct {
	logger logrus.FieldLogger
	ts     *tsnet.Server
}

var _ Connection = (*InProcess)(nil)

// NewInProcess configures a new in-process connection
func NewInProcess(dir string, controlUrl string, tags []string, credentials CredentialProvider) Connection {
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

	return &InProcess{
		logger: logger,
		ts:     ts,
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
