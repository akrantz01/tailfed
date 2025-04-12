package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
)

var cfg config

type config struct {
	LogLevel string `koanf:"log-level"`
	Address  string `koanf:"address"`

	Signing   signingConfig   `koanf:"signing"`
	Storage   storageConfig   `koanf:"storage"`
	Tailscale tailscaleConfig `koanf:"tailscale"`
}

func (c *config) LoadAWSConfig() (aws.Config, error) {
	if c.Signing.Backend == "kms" {
		return awsconfig.LoadDefaultConfig(context.Background())
	}

	return aws.Config{}, nil
}

func (c *config) Validate() error {
	if err := c.Signing.Validate(); err != nil {
		return fmt.Errorf("signing configuration is invalid: %w", err)
	}

	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage configuration is invalid: %w", err)
	}

	if err := c.Tailscale.Validate(); err != nil {
		return fmt.Errorf("tailscale configuration is invalid: %w", err)
	}

	return nil
}

type signingConfig struct {
	Backend  string        `koanf:"backend"`
	Validity time.Duration `koanf:"validity"`
	Key      string        `koanf:"key"`
	Audience string        `koanf:"audience"`
}

func (s *signingConfig) Validate() error {
	if len(s.Audience) == 0 {
		return errors.New("token audience cannot be empty")
	}

	if s.Validity <= 0 {
		return errors.New("token validity must be positive")
	}

	if s.Backend == "kms" && len(s.Key) == 0 {
		return errors.New("missing key for kms backend")
	}

	return nil
}

func (s *signingConfig) NewBackend(config aws.Config) (signing.Backend, error) {
	switch s.Backend {
	case "memory":
		return signing.NewInMemory()
	case "kms":
		return signing.NewKMS(config, s.Key)
	default:
		return nil, errors.New("unknown signing backend")
	}
}

type storageConfig struct {
	Backend string `koanf:"backend"`
	Path    string `koanf:"path"`
}

func (s *storageConfig) Validate() error {
	if s.Backend == "filesystem" && len(s.Path) == 0 {
		return errors.New("missing path for filesystem backend")
	}

	return nil
}

func (s *storageConfig) NewBackend() (storage.Backend, error) {
	switch s.Backend {
	case "filesystem":
		return storage.NewFilesystem(s.Path)
	default:
		return nil, errors.New("unknown storage backend")
	}
}

type tailscaleConfig struct {
	Tailnet string `koanf:"tailnet"`

	ApiKey string               `koanf:"api-key"`
	OAuth  tailscaleOAuthConfig `koanf:"oauth"`
}

func (t *tailscaleConfig) Validate() error {
	if len(t.Tailnet) == 0 {
		return errors.New("a tailnet must be configured")
	}

	if (len(t.ApiKey) > 0) == t.OAuth.Enabled() {
		return errors.New("exactly one tailscale authentication method must be enabled")
	}

	return nil
}

func (t *tailscaleConfig) NewClient() *tailscale.API {
	return tailscale.NewAPI(t.Tailnet, t.Authentication())
}

func (t *tailscaleConfig) Authentication() tailscale.Authentication {
	if len(t.ApiKey) > 0 {
		return tailscale.ApiKey(t.ApiKey)
	}

	return tailscale.OAuth(t.OAuth.ClientId, t.OAuth.ClientSecret)
}

type tailscaleOAuthConfig struct {
	ClientId     string `koanf:"client-id"`
	ClientSecret string `koanf:"client-secret"`
}

func (o *tailscaleOAuthConfig) Enabled() bool {
	return len(o.ClientId) != 0 && len(o.ClientSecret) != 0
}
