package tailscale

import (
	"errors"

	"github.com/sirupsen/logrus"
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

// Authentication determines how the client will authenticate with the Tailscale API
type Authentication interface {
	apply(logger logrus.FieldLogger, c *tailscale.Client)
}

type apiKey struct {
	key string
}

var _ Authentication = (*apiKey)(nil)

// ApiKey authenticates using an unrestricted API key. Where possible, prefer using OAuth credentials.
func ApiKey(key string) Authentication {
	return &apiKey{key}
}

func (a *apiKey) apply(logger logrus.FieldLogger, c *tailscale.Client) {
	logger.Debug("using api key authentication")
	c.APIKey = a.key
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

func (o *oauth) apply(logger logrus.FieldLogger, c *tailscale.Client) {
	logger.Debug("using oauth authentication")
	config := tailscale.OAuthConfig{
		ClientID:     o.id,
		ClientSecret: o.secret,
		Scopes:       []string{""},
	}
	c.HTTP = config.HTTPClient()
}
