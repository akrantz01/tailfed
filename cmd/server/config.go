package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"tailscale.com/tailcfg"
)

var cfg Config

type Config struct {
	LogLevel string `koanf:"log-level" short:"l" help:"The minimum level to log at"`

	Listener  listenerConfig  `koanf:"listener"`
	Signing   signingConfig   `koanf:"signing"`
	Tailscale tailscaleConfig `koanf:"tailscale"`
	Token     tokenConfig     `koanf:"token"`
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

func (c *Config) LoadAWSConfig(ctx context.Context) (aws.Config, error) {
	if c.Signing.Backend.NeedsAWS() {
		return awsconfig.LoadDefaultConfig(ctx)
	}

	return aws.Config{}, nil
}

type listenerConfig struct {
	Address string              `koanf:"address" help:"The host and port the server should bind to"`
	Service tailcfg.ServiceName `koanf:"service" help:"The Tailscale service to advertise"`
}

func (c *listenerConfig) Validate() error {
	hasAddress := len(c.Address) > 0
	hasService := len(c.Service) > 0
	if !hasAddress && !hasService {
		return errors.New("a service or address is required")
	}

	if hasAddress {
		if _, _, err := net.SplitHostPort(c.Address); err != nil {
			return fmt.Errorf("invalid address: %w", err)
		}
	}
	if hasService {
		if err := c.Service.Validate(); err != nil {
			return err
		}
	}

	return nil
}

type signingConfig struct {
	Backend signing.BackendId   `koanf:"backend" help:"How to sign ID tokens"`
	AwsKms  signingAwsKmsConfig `koanf:"aws-kms"`
}

func (c *signingConfig) Validate() error {
	switch c.Backend {
	case signing.BackendMemory:
		return nil

	case signing.BackendAWSKMS:
		return c.AwsKms.Validate()

	default:
		return fmt.Errorf("unknown signing backend %q (options: %s)", c.Backend, signing.BackendIds)
	}
}

func (c *signingConfig) NewBackend(awsConfig aws.Config) (signing.Backend, error) {
	switch c.Backend {
	case signing.BackendMemory:
		return signing.NewInMemory()
	case signing.BackendAWSKMS:
		return signing.NewAWSKMS(awsConfig, c.AwsKms.Key)
	default:
		panic("unreachable")
	}
}

type signingAwsKmsConfig struct {
	Key string `koanf:"key" help:"The AWS KMS key ARN to use"`
}

func (c *signingAwsKmsConfig) Validate() error {
	if len(c.Key) == 0 {
		return errors.New("a KMS key ARN is required")
	}
	return nil
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

type tokenConfig struct {
	Validity time.Duration `koanf:"validity" help:"How long generated tokens should be valid for"`
}

func (c *tokenConfig) Validate() error {
	if c.Validity <= 0 {
		return errors.New("token validity must be positive")
	}

	return nil
}
