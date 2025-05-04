package tailscale

import (
	"context"
	"errors"

	"google.golang.org/grpc/credentials"
	"tailscale.com/client/tailscale/v2"
)

// TLSMode determines how the headscale control plane client should establish its connection
type TLSMode string

const (
	// TLSModeNone uses a plaintext connection
	TLSModeNone TLSMode = "none"
	// TLSModeInsecure uses TLS but does not verify certificates
	TLSModeInsecure TLSMode = "insecure"
	// TLSModeFull uses TLS and verifies certificates
	TLSModeFull TLSMode = "full"
)

var ErrUnknownTLSMode = errors.New("unknown TLS security mode")

// AuthKind is a unique identifier for an Authentication implementation
type AuthKind string

const (
	// AuthKindApiKey is used for the ApiKey authentication
	AuthKindApiKey AuthKind = "api key"
	// AuthKindOAuth is used for the OAuth authentication
	AuthKindOAuth AuthKind = "oauth"
)

// Authentication determines how the client will authenticate with the Tailscale API
type Authentication interface {
	credentials.PerRPCCredentials

	Kind() AuthKind
	tailscale(c *tailscale.Client)
}

type apiKey struct {
	key string
}

var _ Authentication = (*apiKey)(nil)

// ApiKey authenticates using an unrestricted API key. Where possible, prefer using OAuth credentials.
func ApiKey(key string) Authentication {
	return &apiKey{key}
}

func (a *apiKey) Kind() AuthKind {
	return AuthKindApiKey
}

func (a *apiKey) tailscale(c *tailscale.Client) {
	c.APIKey = a.key
}

func (a *apiKey) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + a.key}, nil
}

func (a *apiKey) RequireTransportSecurity() bool {
	return false
}

type oauth struct {
	id     string
	secret string
}

var _ Authentication = (*oauth)(nil)

// OAuth authenticates using temporary credentials derived from a client ID and secret. These are limited in scope.
func OAuth(clientId, clientSecret string) Authentication {
	return &oauth{clientId, clientSecret}
}

func (o *oauth) Kind() AuthKind {
	return AuthKindOAuth
}

func (o *oauth) tailscale(c *tailscale.Client) {
	config := tailscale.OAuthConfig{
		ClientID:     o.id,
		ClientSecret: o.secret,
		Scopes:       []string{""},
	}
	c.HTTP = config.HTTPClient()
}

func (o *oauth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return nil, nil
}

func (o *oauth) RequireTransportSecurity() bool {
	return false
}
