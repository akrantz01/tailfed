package main

import (
	"time"

	"github.com/akrantz01/tailfed/internal/configloader"
	"github.com/akrantz01/tailfed/internal/finalizer"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/akrantz01/tailfed/internal/storage"
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

	// TODO: replace with kms-backed implementation
	var signer signing.Backend = nil

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
	Validity time.Duration `koanf:"validity"`
}
