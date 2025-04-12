package main

import (
	"context"
	"time"

	"github.com/akrantz01/tailfed/internal/configloader"
	"github.com/akrantz01/tailfed/internal/finalizer"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/aws/aws-lambda-go/lambda"
	aws "github.com/aws/aws-sdk-go-v2/config"
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

	awsConfig, err := aws.LoadDefaultConfig(context.Background())
	if err != nil {
		logrus.WithError(err).Fatal("failed to load AWS config from environment")
	}

	signer, err := signing.NewKMS(awsConfig, config.Signing.Key)
	if err != nil {
		logrus.WithError(err).Fatal("failed to initialize signer")
	}

	// TODO: replace with dynamodb-backed implementation
	var store storage.Backend = nil

	handler := finalizer.New(config.Signing.Audience, config.Signing.Validity, signer, store)
	lambda.Start(handler)
}

type Config struct {
	LogLevel string `koanf:"log-level"`

	Signing Signing `koanf:"signing"`
}

type Signing struct {
	Audience string        `koanf:"audience"`
	Key      string        `koanf:"key"`
	Validity time.Duration `koanf:"validity"`
}
