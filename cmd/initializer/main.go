package main

import (
	"context"
	"errors"

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

type Launcher struct {
	StateMachine string `koanf:"state-machine"`
}

type Tailscale struct {
	Backend               string `koanf:"backend"`
	BaseUrl               string `koanf:"base-url"`
	Tailnet               string `koanf:"tailnet"`
	ApiKey                string `koanf:"api-key"`
	OAuthClientId         string `koanf:"oauth-client-id"`
	OAuthClientSecret     string `koanf:"oauth-client-secret"`
	SkipCertificateVerify bool   `koanf:"skip-certificate-verify"`
}

func (t *Tailscale) Client() (tailscale.ControlPlane, error) {
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
		if t.Backend == "headscale" {
			return nil, errors.New("oauth-based authentication unsupported for headscale control plane")
		}

		auth = tailscale.OAuth(t.OAuthClientId, t.OAuthClientSecret)
	}

	if len(t.BaseUrl) == 0 {
		t.BaseUrl = "https://api.tailscale.com"
	}

	switch t.Backend {
	case "hosted":
		return tailscale.NewHostedControlPlane(t.BaseUrl, t.Tailnet, auth)
	case "headscale":
		return tailscale.NewHeadscaleControlPlane(t.Backend, t.Tailnet, t.ApiKey, t.SkipCertificateVerify)
	default:
		return nil, errors.New("unknown tailscale backend")
	}
}

type Storage struct {
	Table string `koanf:"table"`
}
