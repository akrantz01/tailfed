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

	signer, err := signing.NewKMS(logrus.WithField("component", "signer"), awsConfig, config.Signing.Key)
	if err != nil {
		logrus.WithError(err).Fatal("failed to initialize signer")
	}

	store, err := storage.NewDynamo(logrus.WithField("component", "storage"), awsConfig, config.Storage.Table)
	if err != nil {
		logrus.WithError(err).Fatal("failed to initialize store")
	}

	handler := finalizer.New(config.Signing.Audience, config.Signing.Validity, signer, store)
	lambda.Start(handler.Serve)
}

type Config struct {
	LogLevel string `koanf:"log-level"`

	Signing Signing `koanf:"signing"`
	Storage Storage `koanf:"storage"`
}

type Signing struct {
	Audience string        `koanf:"audience"`
	Key      string        `koanf:"key"`
	Validity time.Duration `koanf:"validity"`
}

type Storage struct {
	Table string `koanf:"table"`
}
