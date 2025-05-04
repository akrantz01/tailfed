package main

import (
	"context"
	"errors"
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
	"tailscale.com/tsnet"
)

var ts *tsnet.Server

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

	if err := connectToTailscale(config.Tailscale.AuthKey); err != nil {
		logrus.WithError(err).Fatal("failed to connect to tailscale")
	}

	store, err := storage.NewDynamo(logrus.WithField("component", "storage"), awsConfig, config.Storage.Table)
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

func (c *Config) Validate() error {
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("invaild storage config: %w", err)
	}

	if err := c.Tailscale.Validate(); err != nil {
		return fmt.Errorf("invalid tailscale config: %w", err)
	}

	return nil
}

type Tailscale struct {
	AuthKey string `koanf:"auth-key"`
	Tailnet string `koanf:"tailnet"`
}

func (t *Tailscale) Validate() error {
	if len(t.AuthKey) == 0 {
		return errors.New("missing node auth key")
	}

	if len(t.Tailnet) == 0 {
		return errors.New("missing tailnet name")
	}

	return nil
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

func connectToTailscale(authKey string) error {
	if ts != nil {
		return nil
	}

	logger := logrus.WithField("component", "tailscale")

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("could not get hostname: %w", err)
	}
	tsHostname := fmt.Sprintf("%s-%s-%s", os.Getenv("AWS_LAMBDA_FUNCTION_NAME"), os.Getenv("AWS_REGION"), hostname)
	logger.WithField("hostname", hostname).Info("determined hostname")

	ts = &tsnet.Server{
		AuthKey:   authKey,
		Ephemeral: true,
		Hostname:  tsHostname,
		Dir:       "/tmp",
		UserLogf:  logger.Infof,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	status, err := ts.Up(ctx)
	if err != nil {
		return err
	}

	fields := map[string]any{"status": status.BackendState}
	if status.CurrentTailnet != nil {
		fields["tailnet"] = status.CurrentTailnet.Name
	}
	if status.Self != nil {
		fields["id"] = status.Self.ID
		fields["ips"] = status.Self.TailscaleIPs
	}

	logger.WithFields(fields).Info("successfully connected to tailscale")
	return nil
}
