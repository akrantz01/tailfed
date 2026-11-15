package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"

	"github.com/akrantz01/tailfed/internal/oidc"
	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/aws/aws-sdk-go-v2/aws"
	"tailscale.com/ipn"
)

var cfg Config

type Config struct {
	LogLevel string `koanf:"log-level" short:"l" help:"The minimum level to log at"`

	Listener  listenerConfig  `koanf:"listener"`
	Signing   signingConfig   `koanf:"signing"`
	Tailscale tailscaleConfig `koanf:"tailscale"`
	Token     tokenConfig     `koanf:"token"`
}

func (c *Config) SetDefaults() {
	c.LogLevel = "info"
}

func (c *Config) Validate() error {
	if err := c.Listener.Validate(); err != nil {
		return fmt.Errorf("invalid listener configuration: %w", err)
	}
	if err := c.Signing.Validate(); err != nil {
		return fmt.Errorf("invalid signing configuration: %w", err)
	}
	if err := c.Tailscale.Validate(); err != nil {
		return fmt.Errorf("invalid tailscale configuration: %w", err)
	}
	if err := c.Token.Validate(); err != nil {
		return fmt.Errorf("invalid token configuration: %w", err)
	}

	return nil
}

type listenerConfig struct {
	HTTP   string `koanf:"http" help:"The port (with an optional host) the http listener should bind to"`
	HTTPS  string `koanf:"https" help:"The port (and optional host) the https listener should bind to"`
	Funnel bool   `koanf:"funnel" help:"Expose the token verification routes publicly via Tailscale Funnel"`
}

func (c *listenerConfig) SetDefaults() {
	c.HTTP = "127.0.0.1:8000"
}

func (c *listenerConfig) Validate() error {
	if len(c.HTTP) == 0 {
		return errors.New("at least a http listen address is required")
	}
	if _, _, err := net.SplitHostPort(c.HTTP); err != nil {
		return fmt.Errorf("invalid http listen address: %w", err)
	}

	if len(c.HTTPS) != 0 {
		if _, _, err := net.SplitHostPort(c.HTTPS); err != nil {
			return fmt.Errorf("invalid https listen address: %w", err)
		}
	}

	return nil
}

// Issuer builds the token issuer used in the discovery document and embedded in tokens
func (c *listenerConfig) Issuer(dnsName string) string {
	switch {
	case c.Funnel:
		return "https://" + dnsName
	case len(c.HTTPS) != 0:
		return "https://" + hostWithPort(dnsName, c.HTTPS, "443")
	default:
		return "http://" + hostWithPort(dnsName, c.HTTP, "80")
	}
}

func hostWithPort(name, address, defaultPort string) string {
	_, port, _ := net.SplitHostPort(address)
	if port == defaultPort {
		return name
	}
	return net.JoinHostPort(name, port)
}

type signingConfig struct {
	Backend signing.BackendId   `koanf:"backend" help:"How to sign ID tokens"`
	AwsKms  signingAwsKmsConfig `koanf:"aws-kms"`
	Memory  signingMemoryConfig `koanf:"memory"`
}

func (c *signingConfig) SetDefaults() {
	c.Backend = "memory"
	c.Memory.RS256 = true
	c.Memory.ES256 = true
}

func (c *signingConfig) Validate() error {
	switch c.Backend {
	case signing.BackendMemory:
		return c.Memory.Validate()

	case signing.BackendAWSKMS:
		return c.AwsKms.Validate()

	default:
		return fmt.Errorf("unknown signing backend %q (options: %s)", c.Backend, signing.BackendIds)
	}
}

func (c *signingConfig) NewBackend(awsConfig aws.Config) (signing.Backend, error) {
	switch c.Backend {
	case signing.BackendMemory:
		return signing.NewInMemory(c.Memory.Algorithms())
	case signing.BackendAWSKMS:
		return signing.NewAWSKMS(awsConfig, c.AwsKms.Keys)
	default:
		panic("unreachable")
	}
}

type signingAwsKmsConfig struct {
	Keys []string `koanf:"keys" help:"The AWS KMS key ARN to use"`
}

func (c *signingAwsKmsConfig) Validate() error {
	if len(c.Keys) == 0 {
		return errors.New("at least one KMS key ARN/alias is required")
	}
	return nil
}

type signingMemoryConfig struct {
	RS256 bool `koanf:"rs256" help:"Generate a RSA 2048-bit key"`
	RS384 bool `koanf:"rs384" help:"Generate a RSA 3072-bit key"`
	RS512 bool `koanf:"rs512" help:"Generate a RSA 4096-bit key"`
	ES256 bool `koanf:"es256" help:"Generate an ECDSA curve P-256 key"`
	ES384 bool `koanf:"es384" help:"Generate an ECDSA curve P-384 key"`
	ES512 bool `koanf:"es512" help:"Generate an ECDSA curve P-521 key"`
}

func (c *signingMemoryConfig) Validate() error {
	if c.RS256 || c.RS384 || c.RS512 || c.ES256 || c.ES384 || c.ES512 {
		return nil
	} else {
		return errors.New("at least one signing algorithm type must be enabled")
	}
}

func (c *signingMemoryConfig) Algorithms() map[signing.Algorithm]bool {
	return map[signing.Algorithm]bool{
		signing.RS256: c.RS256,
		signing.RS384: c.RS384,
		signing.RS512: c.RS512,
		signing.ES256: c.ES256,
		signing.ES384: c.ES384,
		signing.ES512: c.ES512,
	}
}

type tailscaleConfig struct {
	Connection tailscale.ConnectionId `koanf:"connection" help:"How to connect to Tailscale"`

	ControlPlaneUrl string                         `koanf:"control-plane-url" help:"The Tailscale/Headscale control plane to connect to (in-process only)"`
	Tags            []string                       `koanf:"tags" help:"The tags to register for the node (in-process only)"`
	Authentication  tailscale.AuthenticationMethod `koanf:"authentication"`

	AuthKey    string                    `koanf:"auth-key" help:"The auth key used to join the tailnet (in-process only)"`
	OAuth      tailscaleOauthConfig      `koanf:"oauth"`
	Federation tailscaleFederationConfig `koanf:"federation"`
}

func (c *tailscaleConfig) SetDefaults() {
	c.Connection = tailscale.ConnectionLocal
	c.ControlPlaneUrl = ipn.DefaultControlURL
	c.Authentication = tailscale.AuthenticationMethodKey
}

func (c *tailscaleConfig) Validate() error {
	switch c.Connection {
	case tailscale.ConnectionLocal:
		return nil

	case tailscale.ConnectionInProcess:
		switch c.Authentication {
		case tailscale.AuthenticationMethodKey:
			if len(c.AuthKey) == 0 {
				return errors.New("an authkey is required")
			} else {
				return nil
			}

		case tailscale.AuthenticationMethodOAuth:
			return c.OAuth.Validate()

		case tailscale.AuthenticationMethodFederation:
			return c.Federation.Validate()

		default:
			return fmt.Errorf("unknown authentication method %q (options: %s)", c.Authentication, tailscale.AuthenticationMethods)
		}

	default:
		return fmt.Errorf("unknown connection method %q (options: %s)", c.Connection, tailscale.ConnectionIds)
	}
}

func (c *tailscaleConfig) Connect(ctx context.Context) (tailscale.Connection, error) {
	switch c.Connection {
	case tailscale.ConnectionLocal:
		return tailscale.NewLocal(ctx)
	case tailscale.ConnectionInProcess:
		return tailscale.NewInProcess(ctx, "", c.ControlPlaneUrl, c.Tags, c.credentials())
	default:
		panic("unreachable")
	}
}

func (c *tailscaleConfig) credentials() tailscale.CredentialProvider {
	switch c.Authentication {
	case tailscale.AuthenticationMethodKey:
		return tailscale.AuthKey(c.AuthKey)
	case tailscale.AuthenticationMethodOAuth:
		return tailscale.OAuth(c.OAuth.ClientId, c.OAuth.ClientSecret)
	case tailscale.AuthenticationMethodFederation:
		return tailscale.WorkloadIdentityFederation(c.Federation.ClientId, c.Federation.IdToken, c.Federation.Audience)
	default:
		panic("unreachable")
	}
}

type tailscaleOauthConfig struct {
	ClientId     string `koanf:"client-id" help:"The OAuth app client ID"`
	ClientSecret string `koanf:"client-secret" help:"The OAuth app client secret"`
}

func (c *tailscaleOauthConfig) Validate() error {
	switch {
	case len(c.ClientId) == 0:
		return errors.New("a client ID is required")
	case len(c.ClientSecret) == 0:
		return errors.New("a client secret is required")
	default:
		return nil
	}
}

type tailscaleFederationConfig struct {
	ClientId string `koanf:"client-id" help:"The OpenID connect client ID"`
	IdToken  string `koanf:"id-token" help:"The identity provider's existing ID token"`
	Audience string `koanf:"audience" help:"The audience used to automatically fetch an ID token"`
}

func (c *tailscaleFederationConfig) Validate() error {
	switch {
	case len(c.ClientId) == 0:
		return errors.New("a client ID is required")
	case len(c.IdToken) == 0 && len(c.Audience) == 0:
		return errors.New("an ID token or audience is required")
	default:
		return nil
	}
}

var profileIdRegex = regexp.MustCompile("^[a-z0-9]+$")

type tokenConfig struct {
	DefaultProfile string                  `koanf:"default-profile" help:"The default profile to use when none is requested"`
	Profiles       map[string]oidc.Profile `koanf:"profiles"`
}

func (c *tokenConfig) SetDefaults() {
	var p oidc.Profile
	p.SetDefaults()

	c.DefaultProfile = "default"
	c.Profiles = map[string]oidc.Profile{c.DefaultProfile: p}
}

func (c *tokenConfig) Validate() error {
	for id, profile := range c.Profiles {
		if !profileIdRegex.MatchString(id) {
			return fmt.Errorf("profile id %q must contain only lowercase alphanumeric characters", id)
		}

		if err := profile.Validate(); err != nil {
			return fmt.Errorf("invalid profile %q: %w", id, err)
		}
	}

	if _, ok := c.Profiles[c.DefaultProfile]; !ok {
		return fmt.Errorf("default profile %q must exist in profiles", c.DefaultProfile)
	}

	return nil
}
