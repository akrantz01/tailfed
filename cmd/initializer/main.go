package main

import (
	"errors"

	"github.com/akrantz01/tailfed/internal/configloader"
	"github.com/akrantz01/tailfed/internal/initializer"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
)

func main() {
	var config Config
	if err := configloader.LoadInto(&config, configloader.WithEnvPrefix("TAILFED_")); err != nil {
		logrus.WithError(err).Fatal("failed to load configuration")
	}

	if err := logging.Initialize(config.LogLevel); err != nil {
		logrus.WithError(err).Fatal("failed to initialize logging")
	}

	tsClient, err := config.Tailscale.Client()
	if err != nil {
		logrus.WithError(err).Fatal("failed to create tailscale client")
	}

	// TODO: replace with dynamodb-backed implementation
	var store storage.Backend = nil

	handler := initializer.New(tsClient, store)
	lambda.Start(handler.Serve)
}

type Config struct {
	LogLevel  string    `koanf:"log-level"`
	Tailscale Tailscale `koanf:"tailscale"`
}

type Tailscale struct {
	Tailnet           string `koanf:"tailnet"`
	ApiKey            string `koanf:"api-key"`
	OAuthClientId     string `koanf:"oauth-client-id"`
	OAuthClientSecret string `koanf:"oauth-client-secret"`
}

func (t *Tailscale) Client() (*tailscale.API, error) {
	if len(t.Tailnet) == 0 {
		return nil, errors.New("missing tailnet name")
	}

	apiKeyEnabled := len(t.ApiKey) != 0
	oauthEnabled := len(t.OAuthClientId) != 0 && len(t.OAuthClientSecret) != 0
	if apiKeyEnabled == oauthEnabled {
		return nil, errors.New("exactly one tailscale authentication method must be configured")
	}

	var auth tailscale.Authentication
	if apiKeyEnabled {
		auth = tailscale.ApiKey(t.ApiKey)
	} else {
		auth = tailscale.OAuth(t.OAuthClientId, t.OAuthClientSecret)
	}

	return tailscale.NewAPI(t.Tailnet, auth), nil
}
