package main

import (
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/akrantz01/tailfed/internal/configloader"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/akrantz01/tailfed/internal/verifier"
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

	proxyUrl, err := url.Parse(os.Getenv("ALL_PROXY"))
	if err != nil {
		logrus.WithError(err).Fatal("failed to parse tailscale proxy url")
	}

	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)},
		Timeout:   15 * time.Second,
	}

	// TODO: replace with dynamodb-backed implementation
	var store storage.Backend = nil

	handler := verifier.New(client, store, config.Tailscale.Tailnet)
	lambda.Start(handler.Serve)
}

type Config struct {
	LogLevel string `koanf:"log-level"`

	Tailscale Tailscale `koanf:"tailscale"`
}

type Tailscale struct {
	Tailnet string `koanf:"tailnet"`
}
