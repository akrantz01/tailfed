package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/akrantz01/tailfed/internal/configloader"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/akrantz01/tailfed/internal/verifier"
	"github.com/aws/aws-lambda-go/lambda"
	aws "github.com/aws/aws-sdk-go-v2/config"
	"github.com/sirupsen/logrus"
	"tailscale.com/hostinfo"
	"tailscale.com/tsnet"
)

var ts *tsnet.Server

func main() {
	var config Config
	if err := configloader.LoadInto(&config, configloader.WithEnvPrefix("TAILFED_")); err != nil {
		logrus.WithError(err).Fatal("failed to load configuration")
	}

	if err := logging.Initialize(config.LogLevel); err != nil {
		logrus.WithError(err).Fatal("failed to initialize logging")
	}

	if err := connectToTailscale(config.Tailscale.AuthKey); err != nil {
		logrus.WithError(err).Fatal("failed to connect to tailscale")
	}

	awsConfig, err := aws.LoadDefaultConfig(context.Background())
	if err != nil {
		logrus.WithError(err).Fatal("failed to load AWS config from environment")
	}

	store, err := storage.NewDynamo(awsConfig, config.Storage.Table)
	if err != nil {
		logrus.WithError(err).Fatal("failed to initialize store")
	}

	client := ts.HTTPClient()
	client.Timeout = 5 * time.Second

	handler := verifier.New(client, store, config.Tailscale.Tailnet)
	lambda.Start(handler.Serve)
}

type Config struct {
	LogLevel string `koanf:"log-level"`

	Storage   Storage   `koanf:"storage"`
	Tailscale Tailscale `koanf:"tailscale"`
}

type Tailscale struct {
	AuthKey string `koanf:"auth-key"`
	Tailnet string `koanf:"tailnet"`
}

type Storage struct {
	Table string `koanf:"table"`
}

func connectToTailscale(authKey string) error {
	if ts != nil {
		return nil
	}

	hostinfo.SetApp("tailfed-verifier")

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("could not get hostname: %w", err)
	}

	ts = &tsnet.Server{
		AuthKey:   authKey,
		Ephemeral: true,
		Hostname:  fmt.Sprintf("tailfed-initializer-%s-%s", os.Getenv("AWS_LAMBDA_FUNCTION_VERSION"), hostname),
		Dir:       "/tmp",
	}
	if err := ts.Start(); err != nil {
		ts = nil
		return err
	}

	return nil
}
