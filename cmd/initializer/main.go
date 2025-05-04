package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/akrantz01/tailfed/internal/configloader"
	"github.com/akrantz01/tailfed/internal/initializer"
	"github.com/akrantz01/tailfed/internal/launcher"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/aws/aws-lambda-go/lambda"
	aws "github.com/aws/aws-sdk-go-v2/config"
	"github.com/sirupsen/logrus"
)

func main() {
	awsConfig, err := aws.LoadDefaultConfig(context.Background())
	if err != nil {
		logrus.WithError(err).Fatal("failed to load AWS config from environment")
	}

	var config Config
	if err := configloader.LoadInto(&config, configloader.WithEnvPrefix("TAILFED_"), configloader.WithSecrets(awsConfig)); err != nil {
		logrus.WithError(err).Fatal("failed to load configuration")
	}
	if err := config.Validate(); err != nil {
		logrus.WithError(err).Fatal("invalid configuration")
	}

	if err := logging.Initialize(config.LogLevel); err != nil {
		logrus.WithError(err).Fatal("failed to initialize logging")
	}

	tsClient, err := config.Tailscale.Client()
	if err != nil {
		logrus.WithError(err).Fatal("failed to create tailscale client")
	}

	launch, err := launcher.NewStepFunction(logrus.WithField("component", "launcher"), awsConfig, config.Launcher.StateMachine)
	if err != nil {
		logrus.WithError(err).Fatal("failed to initialize launcher")
	}

	store, err := storage.NewDynamo(logrus.WithField("component", "storage"), awsConfig, config.Storage.Table)
	if err != nil {
		logrus.WithError(err).Fatal("failed to initialize store")
	}

	handler := initializer.New(tsClient, launch, store)
	lambda.Start(handler.Serve)
}

type Config struct {
	LogLevel string `koanf:"log-level"`

	Launcher  Launcher  `koanf:"launcher"`
	Tailscale Tailscale `koanf:"tailscale"`
	Storage   Storage   `koanf:"storage"`
}

func (c *Config) Validate() error {
	if err := c.Launcher.Validate(); err != nil {
		return fmt.Errorf("invalid launcher config: %w", err)
	}

	if err := c.Tailscale.Validate(); err != nil {
		return fmt.Errorf("invalid tailscale config: %w", err)
	}

	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("invalid storage config: %w", err)
	}

	return nil
}

type Launcher struct {
	StateMachine string `koanf:"state-machine"`
}

func (l *Launcher) Validate() error {
	if len(l.StateMachine) == 0 {
		return errors.New("missing state machine identifier")
	}

	return nil
}

type Tailscale struct {
	Backend string `koanf:"backend"`
	BaseUrl string `koanf:"base-url"`

	Tailnet string `koanf:"tailnet"`

	ApiKey            string            `koanf:"api-key"`
	OAuthClientId     string            `koanf:"oauth-client-id"`
	OAuthClientSecret string            `koanf:"oauth-client-secret"`
	TLSMode           tailscale.TLSMode `koanf:"tls-mode"`

	auth tailscale.Authentication
}

func (t *Tailscale) Validate() error {
	if len(t.BaseUrl) == 0 {
		t.BaseUrl = "https://api.tailscale.com"
	}
	if baseUrl, err := url.Parse(t.BaseUrl); err != nil {
		return fmt.Errorf("invalid base url: %w", err)
	} else if baseUrl.Scheme != "http" && baseUrl.Scheme != "https" {
		return errors.New("base url scheme must be http or https")
	}

	if len(t.Tailnet) == 0 {
		return errors.New("missing tailnet name")
	}

	apiKeyEnabled := len(t.ApiKey) != 0
	oauthEnabled := len(t.OAuthClientId) != 0 && len(t.OAuthClientSecret) != 0
	if apiKeyEnabled == oauthEnabled {
		return errors.New("exactly one authentication method must be configured")
	}

	if apiKeyEnabled {
		t.auth = tailscale.ApiKey(t.ApiKey)
	} else {
		t.auth = tailscale.OAuth(t.OAuthClientId, t.OAuthClientSecret)
	}

	return nil
}

func (t *Tailscale) Client() (tailscale.ControlPlane, error) {
	logger := logrus.WithField("component", "tailscale")
	return tailscale.NewControlPlane(logger, t.Backend, t.BaseUrl, t.Tailnet, t.auth, t.TLSMode)
}

type Storage struct {
	Table string `koanf:"table"`
}

func (s *Storage) Validate() error {
	if len(s.Table) == 0 {
		return errors.New("missing DynamoDB table name")
	}

	return nil
}
