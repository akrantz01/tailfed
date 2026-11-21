package tailscale

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"

	"github.com/sirupsen/logrus"
	"tailscale.com/client/local"
	_ "tailscale.com/feature/identityfederation"
	"tailscale.com/tsnet"
)

// AuthenticationMethod determines how tailfed should authenticate for in-process connections
type AuthenticationMethod string

const (
	// AuthenticationMethodKey uses a static authkey
	AuthenticationMethodKey AuthenticationMethod = "authkey"
	// AuthenticationMethodOAuth uses the OAuth flow for generating authkey(s)
	AuthenticationMethodOAuth AuthenticationMethod = "oauth"
	// AuthenticationMethodFederation uses workload identity federation for generating authkey(s)
	AuthenticationMethodFederation AuthenticationMethod = "federation"
)

func (a AuthenticationMethod) Choices() []string {
	return []string{
		string(AuthenticationMethodKey),
		string(AuthenticationMethodOAuth),
		string(AuthenticationMethodFederation),
	}
}

var AuthenticationMethods string = strings.Join(AuthenticationMethodKey.Choices(), ", ")

// InProcess uses tsnet to connect to Tailscale for the current process only
type InProcess struct {
	logger logrus.FieldLogger
	ts     *tsnet.Server
	client *local.Client
}

var _ Connection = (*InProcess)(nil)

// NewInProcess configures a new in-process connection
func NewInProcess(ctx context.Context, hostname, dir, controlUrl string, tags []string, credentials CredentialProvider) (Connection, error) {
	logger := logrus.
		WithField("component", "tailscale").
		WithField("tailscale.connection", "in-process")

	ts := &tsnet.Server{
		Dir:           dir,
		Hostname:      hostname,
		ControlURL:    controlUrl,
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

func (p *InProcess) ListenPrivate(_ context.Context, port uint16) (net.Listener, error) {
	return p.ts.Listen("tcp", fmt.Sprintf(":%d", port))
}

func (p *InProcess) ListenFunnel(_ context.Context) (net.Listener, error) {
	return p.ts.ListenFunnel("tcp", ":443", tsnet.FunnelOnly())
}

func (p *InProcess) DNSName(ctx context.Context) (string, error) {
	status, err := p.client.StatusWithoutPeers(ctx)
	if err != nil {
		return "", err
	}
	if status.Self == nil || status.Self.DNSName == "" {
		return "", errors.New("node has no dns name")
	}

	return strings.TrimSuffix(status.Self.DNSName, "."), nil
}

func (p *InProcess) GetCertificate(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return p.client.GetCertificate(hi)
}

func (p *InProcess) WhoIs(ctx context.Context, address string) (*Node, error) {
	status, err := p.client.StatusWithoutPeers(ctx)
	if err != nil {
		return nil, err
	}

	res, err := p.client.WhoIs(ctx, address)
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
