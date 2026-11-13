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
	LogLevel string `koanf:"log-level"`

	Listener  listenerConfig  `koanf:"listener"`
	Signing   signingConfig   `koanf:"signing"`
	Tailscale tailscaleConfig `koanf:"tailscale"`
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

	return nil
}

func (c *Config) LoadAWSConfig(ctx context.Context) (aws.Config, error) {
	if c.Signing.Backend.NeedsAws() {
		return awsconfig.LoadDefaultConfig(ctx)
	}

	return aws.Config{}, nil
}

type listenerConfig struct {
	Address string              `koanf:"address"`
	Service tailcfg.ServiceName `koanf:"service"`
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

// SigningBackend determines how tailfed should sign ID tokens
type SigningBackend string

const (
	// SigningBackendMemory uses ephemeral in-memory keys
	SigningBackendMemory SigningBackend = "memory"
	// SigningBackendAwsKms uses an AWS KMS-based key
	SigningBackendAwsKms SigningBackend = "aws-kms"
)

func (s SigningBackend) NeedsAws() bool {
	return s == SigningBackendAwsKms
}

type signingConfig struct {
	Backend  SigningBackend      `koanf:"backend"`
	Validity time.Duration       `koanf:"validity"`
	AwsKms   signingAwsKmsConfig `koanf:"aws-kms"`
}

func (c *signingConfig) Validate() error {
	if c.Validity <= 0 {
		return errors.New("token validity must be positive")
	}

	switch c.Backend {
	case SigningBackendMemory:
		return nil

	case SigningBackendAwsKms:
		return c.AwsKms.Validate()

	default:
		return fmt.Errorf("unknown signing backend %q (options: %s, %s)", c.Backend, SigningBackendAwsKms, SigningBackendMemory)
	}
}

func (c *signingConfig) NewBackend(awsConfig aws.Config) (signing.Backend, error) {
	switch c.Backend {
	case SigningBackendMemory:
		return signing.NewInMemory()
	case SigningBackendAwsKms:
		return signing.NewAWSKMS(awsConfig, c.AwsKms.Key)
	default:
		panic("unreachable")
	}
}

type signingAwsKmsConfig struct {
	Key string `koanf:"key"`
}

func (c *signingAwsKmsConfig) Validate() error {
	if len(c.Key) == 0 {
		return errors.New("a KMS key ARN is required")
	}
	return nil
}

// TailscaleConnection determines how tailfed should connect to Tailscale
type TailscaleConnection string

const (
	// TailscaleConnectionLocal uses an existing Tailscale daemon running on the host
	TailscaleConnectionLocal TailscaleConnection = "local"
	// TailscaleConnectionInProcess connects to Tailscale for the current process only
	TailscaleConnectionInProcess TailscaleConnection = "in-process"
)

// TailscaleAuthMethod determines how tailfed should authenticate with Tailscale for in-process connections
type TailscaleAuthMethod string

const (
	// TailscaleAuthMethodAuthKey uses a static authkey
	TailscaleAuthMethodAuthKey TailscaleAuthMethod = "auth-key"
	// TailscaleAuthMethodOAuth uses the OAuth flow for generating authkey(s)
	TailscaleAuthMethodOAuth TailscaleAuthMethod = "oauth"
	// TailscaleAuthMethodFederation uses workload identity federation for generating authkey(s)
	TailscaleAuthMethodFederation TailscaleAuthMethod = "federation"
)

type tailscaleConfig struct {
	Connection TailscaleConnection `koanf:"connection"`

	ControlPlaneUrl string              `koanf:"control-plane-url"`
	Tags            []string            `koanf:"tags"`
	Authentication  TailscaleAuthMethod `koanf:"authentication"`

	AuthKey    string                    `koanf:"auth-key"`
	OAuth      tailscaleOauthConfig      `koanf:"oauth"`
	Federation tailscaleFederationConfig `koanf:"federation"`
}

func (c *tailscaleConfig) Validate() error {
	switch c.Connection {
	case TailscaleConnectionLocal:
		return nil

	case TailscaleConnectionInProcess:
		switch c.Authentication {
		case TailscaleAuthMethodAuthKey:
			if len(c.AuthKey) == 0 {
				return errors.New("an authkey is required")
			} else {
				return nil
			}

		case TailscaleAuthMethodOAuth:
			return c.OAuth.Validate()

		case TailscaleAuthMethodFederation:
			return c.Federation.Validate()

		default:
			return fmt.Errorf("unknown authentication method %q (options: %s, %s, %s)", c.Authentication, TailscaleAuthMethodAuthKey, TailscaleAuthMethodOAuth, TailscaleAuthMethodFederation)
		}

	default:
		return fmt.Errorf("unknown connection method %q (options: %s, %s)", c.Connection, TailscaleConnectionLocal, TailscaleConnectionInProcess)
	}
}

func (c *tailscaleConfig) Connect(ctx context.Context) (tailscale.Connection, error) {
	switch c.Connection {
	case TailscaleConnectionLocal:
		return tailscale.NewLocal(ctx)
	case TailscaleConnectionInProcess:
		return tailscale.NewInProcess(ctx, "", c.ControlPlaneUrl, c.Tags, c.credentials())
	default:
		panic("unreachable")
	}
}

func (c *tailscaleConfig) credentials() tailscale.CredentialProvider {
	switch c.Authentication {
	case TailscaleAuthMethodAuthKey:
		return tailscale.AuthKey(c.AuthKey)
	case TailscaleAuthMethodOAuth:
		return tailscale.OAuth(c.OAuth.ClientId, c.OAuth.ClientSecret)
	case TailscaleAuthMethodFederation:
		return tailscale.WorkloadIdentityFederation(c.Federation.ClientId, c.Federation.IdToken, c.Federation.Audience)
	default:
		panic("unreachable")
	}
}

type tailscaleOauthConfig struct {
	ClientId     string `koanf:"client-id"`
	ClientSecret string `koanf:"client-secret"`
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
	ClientId string `koanf:"client-id"`
	IdToken  string `koanf:"id-token"`
	Audience string `koanf:"audience"`
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
