package main

import (
	"errors"
	"fmt"
	"net"

	"github.com/akrantz01/tailfed/internal/tailscale"
)

var cfg Config

type Config struct {
	LogLevel string `koanf:"log-level"`

	Listener  listenerConfig  `koanf:"listener"`
	Tailscale tailscaleConfig `koanf:"tailscale"`
}

func (c *Config) Validate() error {
	if err := c.Listener.Validate(); err != nil {
		return fmt.Errorf("invalid listener configuration: %w", err)
	}
	if err := c.Tailscale.Validate(); err != nil {
		return fmt.Errorf("invalid tailscale configuration: %w", err)
	}

	return nil
}

type listenerConfig struct {
	Address string `koanf:"address"`
	Service string `koanf:"service"`
}

func (c *listenerConfig) Validate() error {
	if len(c.Address) > 0 {
		if _, _, err := net.SplitHostPort(c.Address); err != nil {
			return fmt.Errorf("invalid address: %w", err)
		}
	} else if len(c.Service) == 0 {
		return errors.New("a service or address must be configured")
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
	// TailscaleAuthMethodWorkload uses workload identity federation for generating authkey(s)
	TailscaleAuthMethodWorkload TailscaleAuthMethod = "workload"
)

type tailscaleConfig struct {
	Connection TailscaleConnection `koanf:"connection"`

	ControlPlaneUrl string              `koanf:"control-plane-url"`
	Tags            []string            `koanf:"tags"`
	Authentication  TailscaleAuthMethod `koanf:"authentication"`

	AuthKey      string `koanf:"auth-key"`
	ClientId     string `koanf:"client-id"`
	ClientSecret string `koanf:"client-secret"`
	IdToken      string `koanf:"id-token"`
	Audience     string `koanf:"audience"`
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
			}
		case TailscaleAuthMethodOAuth:
			if len(c.ClientId) == 0 {
				return errors.New("a client ID is required")
			} else if len(c.ClientSecret) == 0 {
				return errors.New("a client secret is required")
			}

		case TailscaleAuthMethodWorkload:
			if len(c.ClientId) == 0 {
				return errors.New("a client ID is required")
			} else if len(c.IdToken) == 0 && len(c.Audience) == 0 {
				return errors.New("an ID token or audience is required")
			}

		default:
			return fmt.Errorf("unknown authentication method '%s' (options: %s, %s, %s)", c.Audience, TailscaleAuthMethodAuthKey, TailscaleAuthMethodOAuth, TailscaleAuthMethodWorkload)
		}

		return nil

	default:
		return fmt.Errorf("unknown connection method '%s' (options: %s, %s)", c.Connection, TailscaleConnectionLocal, TailscaleConnectionInProcess)
	}
}

func (c *tailscaleConfig) Connect() tailscale.Connection {
	switch c.Connection {
	case TailscaleConnectionLocal:
		return tailscale.NewLocal()
	case TailscaleConnectionInProcess:
		return tailscale.NewInProcess("", c.ControlPlaneUrl, c.Tags, c.credentials())
	default:
		panic("unreachable")
	}
}

func (c *tailscaleConfig) credentials() tailscale.CredentialProvider {
	switch c.Authentication {
	case TailscaleAuthMethodAuthKey:
		return tailscale.AuthKey(c.AuthKey)
	case TailscaleAuthMethodOAuth:
		return tailscale.OAuth(c.ClientId, c.ClientSecret)
	case TailscaleAuthMethodWorkload:
		return tailscale.WorkloadIdentityFederation(c.ClientId, c.IdToken, c.Audience)
	default:
		panic("unreachable")
	}
}
