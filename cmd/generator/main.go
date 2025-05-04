package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/akrantz01/tailfed/internal/configloader"
	"github.com/akrantz01/tailfed/internal/generator"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/metadata"
	"github.com/akrantz01/tailfed/internal/signing"
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

	meta, err := metadata.NewS3(logrus.WithField("component", "metadata"), awsConfig, config.Metadata.Bucket)
	if err != nil {
		logrus.WithError(err).Fatal("failed to initialize metadata")
	}

	signer, err := signing.NewKMS(logrus.WithField("component", "signer"), awsConfig, config.Signing.Key)
	if err != nil {
		logrus.WithError(err).Fatal("failed to initialize signer")
	}

	handler := generator.New(config.Signing.Validity, meta, signer)
	lambda.Start(handler.Serve)
}

type Config struct {
	LogLevel string `koanf:"log-level"`

	Metadata Metadata `koanf:"metadata"`
	Signing  Signing  `koanf:"signing"`
}

func (c *Config) Validate() error {
	if err := c.Metadata.Validate(); err != nil {
		return fmt.Errorf("invalid metadata config: %w", err)
	}

	if err := c.Signing.Validate(); err != nil {
		return fmt.Errorf("invalid signing config: %w", err)
	}

	return nil
}

type Metadata struct {
	Bucket string `koanf:"bucket"`
}

func (m *Metadata) Validate() error {
	if len(m.Bucket) == 0 {
		return errors.New("missing bucket name")
	}

	return nil
}

type Signing struct {
	Key      string        `koanf:"key"`
	Validity time.Duration `koanf:"validity"`
}

func (s *Signing) Validate() error {
	if len(s.Key) == 0 {
		return errors.New("missing KMS signing key")
	}

	if s.Validity <= 0 {
		return errors.New("token validity must be positive")
	}

	return nil
}
