package tailscale

import "tailscale.com/client/tailscale/v2"

// Authentication determines how the client will authenticate with the Tailscale API
type Authentication interface {
	apply(c *tailscale.Client)
}

type apiKey struct {
	key string
}

var _ Authentication = (*apiKey)(nil)

// ApiKey authenticates using an unrestricted API key. Where possible, prefer using OAuth credentials.
func ApiKey(key string) Authentication {
	return &apiKey{key}
}

func (a *apiKey) apply(c *tailscale.Client) {
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

func (o *oauth) apply(c *tailscale.Client) {
	config := tailscale.OAuthConfig{
		ClientID:     o.id,
		ClientSecret: o.secret,
		Scopes:       []string{""},
	}
	c.HTTP = config.HTTPClient()
}
