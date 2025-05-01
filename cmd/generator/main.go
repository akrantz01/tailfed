package main

import (
	"context"

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

	if err := logging.Initialize(config.LogLevel); err != nil {
		logrus.WithError(err).Fatal("failed to initialize logging")
	}

	meta, err := metadata.NewS3(logrus.WithField("component", "metadata"), awsConfig, config.Metadata.Bucket)
	if err != nil {
		logrus.WithError(err).Fatal("failed to initialize metadata")
	}

	signer, err := signing.NewKMS(awsConfig, config.Signing.Key)
	if err != nil {
		logrus.WithError(err).Fatal("failed to initialize signer")
	}

	handler := generator.New(meta, signer)
	lambda.Start(handler.Serve)
}

type Config struct {
	LogLevel string `koanf:"log-level"`

	Metadata Metadata `koanf:"metadata"`
	Signing  Signing  `koanf:"signing"`
}

type Metadata struct {
	Bucket string `koanf:"bucket"`
}

type Signing struct {
	Key string `koanf:"key"`
}
