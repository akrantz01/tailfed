package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/akrantz01/tailfed/internal/launcher"
	"github.com/akrantz01/tailfed/internal/metadata"
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

	Launcher  launcherConfig  `koanf:"launcher"`
	Metadata  metadataConfig  `koanf:"metadata"`
	Signing   signingConfig   `koanf:"signing"`
	Storage   storageConfig   `koanf:"storage"`
	Tailscale tailscaleConfig `koanf:"tailscale"`
}

func (c *config) LoadAWSConfig() (aws.Config, error) {
	if c.Launcher.Backend == "step-function" ||
		c.Metadata.Bucket == "s3" ||
		c.Signing.Backend == "kms" ||
		c.Storage.Backend == "dynamo" {
		return awsconfig.LoadDefaultConfig(context.Background())
	}

	return aws.Config{}, nil
}

func (c *config) Validate() error {
	if err := c.Launcher.Validate(); err != nil {
		return fmt.Errorf("launcher configuration is invalid: %w", err)
	}

	if err := c.Metadata.Validate(); err != nil {
		return fmt.Errorf("metadata configuration is invalid: %w", err)
	}

	if err := c.Signing.Validate(); err != nil {
		return fmt.Errorf("signing configuration is invalid: %w", err)
	}

	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage configuration is invalid: %w", err)
	}

	if err := c.Tailscale.Validate(); err != nil {
		return fmt.Errorf("tailscale configuration is invalid: %w", err)
	}

	if c.Launcher.Backend == "step-function" && c.Storage.Backend != "dynamo" {
		return errors.New("step-function launcher backend requires dynamo storage backend")
	}

	return nil
}

type launcherConfig struct {
	Backend      string `koanf:"backend"`
	StateMachine string `koanf:"state-machine"`
}

func (l *launcherConfig) Validate() error {
	if l.Backend == "step-function" && len(l.StateMachine) == 0 {
		return errors.New("missing state machine arn for step-function backend")
	}

	return nil
}

func (l *launcherConfig) NewBackend(config aws.Config, bus chan<- launcher.Request) (launcher.Backend, error) {
	switch l.Backend {
	case "local":
		return launcher.NewLocal(bus), nil
	case "step-function":
		return launcher.NewStepFunction(config, l.StateMachine)
	default:
		return nil, errors.New("unknown launcher backend")
	}
}

type metadataConfig struct {
	Backend string `koanf:"backend"`
	Bucket  string `koanf:"bucket"`
	Path    string `koanf:"path"`
}

func (m *metadataConfig) Validate() error {
	if m.Backend == "filesystem" && len(m.Path) == 0 {
		return errors.New("missing path for filesystem backend")
	}

	if m.Backend == "dynamo" && len(m.Backend) == 0 {
		return errors.New("missing bucket for s3 backend")
	}

	return nil
}

func (m *metadataConfig) NewBackend(config aws.Config) (metadata.Backend, error) {
	switch m.Backend {
	case "filesystem":
		return metadata.NewFilesystem(m.Path)
	case "s3":
		return metadata.NewS3(config, m.Bucket)
	default:
		return nil, errors.New("unknown metadata backend")
	}
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
	Table   string `koanf:"table"`
}

func (s *storageConfig) Validate() error {
	if s.Backend == "filesystem" && len(s.Path) == 0 {
		return errors.New("missing path for filesystem backend")
	}

	if s.Backend == "dynamo" && len(s.Table) == 0 {
		return errors.New("missing table for dynamo backend")
	}

	return nil
}

func (s *storageConfig) NewBackend(config aws.Config) (storage.Backend, error) {
	switch s.Backend {
	case "filesystem":
		return storage.NewFilesystem(s.Path)
	case "dynamo":
		return storage.NewDynamo(config, s.Table)
	default:
		return nil, errors.New("unknown storage backend")
	}
}

type tailscaleConfig struct {
	Backend string `koanf:"backend"`
	BaseUrl string `koanf:"base-url"`

	Tailnet string `koanf:"tailnet"`

	ApiKey string               `koanf:"api-key"`
	OAuth  tailscaleOAuthConfig `koanf:"oauth"`
}

func (t *tailscaleConfig) Validate() error {
	if len(t.BaseUrl) == 0 {
		return errors.New("a base url must be configured")
	}

	if len(t.Tailnet) == 0 {
		return errors.New("a tailnet must be configured")
	}

	if (len(t.ApiKey) > 0) == t.OAuth.Enabled() {
		return errors.New("exactly one tailscale authentication method must be enabled")
	}

	if t.Backend == "headscale" && t.OAuth.Enabled() {
		return errors.New("oauth-based authentication not supported by headscale")
	}

	return nil
}

func (t *tailscaleConfig) NewClient() (tailscale.ControlPlane, error) {
	switch t.Backend {
	case "hosted":
		return tailscale.NewHostedControlPlane(t.BaseUrl, t.Tailnet, t.Authentication())
	case "headscale":
		return tailscale.NewHeadscaleControlPlane(t.Backend, t.Tailnet, t.ApiKey)
	default:
		return nil, errors.New("unknown tailscale backend")
	}
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
