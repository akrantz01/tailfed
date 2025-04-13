package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akrantz01/tailfed/internal/configloader"
	"github.com/akrantz01/tailfed/internal/generator"
	"github.com/akrantz01/tailfed/internal/http/gateway"
	"github.com/akrantz01/tailfed/internal/launcher"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:           "dev",
		Short:         "Mimics the API gateway and step function for coordinating the Lambda functions",
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE:       preRun,
		RunE:          run,
	}
	cmd.Flags().StringP("log-level", "l", "info", "The minimum level to log at (choices: panic, fatal, error, warn, info, debug, trace)")
	cmd.Flags().StringP("address", "a", "127.0.0.1:8000", "The address and port combination to listen on")

	cmd.Flags().String("launcher.backend", "local", "Where to launch the verification flow (choices: local, step-function)")
	cmd.Flags().String("launcher.state-machine", "", "The ARN of the state machine to use for the step-function backend")

	cmd.Flags().String("metadata.backend", "filesystem", "Where to store OpenID Connect metadata (choices: filesystem, s3)")
	cmd.Flags().String("metadata.bucket", "", "The bucket to store metadata in for the s3 backend")
	cmd.Flags().String("metadata.path", "metadata", "The directory path used by the filesystem backend")

	cmd.Flags().String("signing.backend", "memory", "The method used to sign JWTs (choices: memory, kms)")
	cmd.Flags().Duration("signing.validity", 1*time.Hour, "How long the generated tokens should be valid for")
	cmd.Flags().String("signing.audience", "sts.amazonaws.com", "The audience the tokens are issued for")

	cmd.Flags().String("storage.backend", "filesystem", "Where to store data for in-flight flows (choices: dynamo, filesystem)")
	cmd.Flags().String("storage.path", "flows", "The directory path used by the filesystem backend")
	cmd.Flags().String("storage.table", "", "The name of the DynamoDB table used by the dynamo backend")

	cmd.Flags().String("tailscale.tailnet", "", "The name of the tailnet to issue tokens for")
	cmd.Flags().String("tailscale.api-key", "", "The Tailscale API key to authenticate with")
	cmd.Flags().String("tailscale.oauth.client-id", "", "The Tailscale OAuth client ID to authenticate with")
	cmd.Flags().String("tailscale.oauth.client-secret", "", "The Tailscale OAuth client secret to authenticate with")

	err := cmd.Execute()
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

// preRun configures the runtime environment
func preRun(cmd *cobra.Command, _ []string) error {
	err := configloader.LoadInto(&cfg,
		configloader.WithFlags(cmd.Flags()),
		configloader.WithEnvPrefix("DEV_GATEWAY_"),
	)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := logging.Initialize(cfg.LogLevel); err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	return nil
}

// run configures and launches the development gateway
func run(*cobra.Command, []string) error {
	awsConfig, err := cfg.LoadAWSConfig()
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	bus := make(chan launcher.Request, 3)
	launch, err := cfg.Launcher.NewBackend(awsConfig, bus)
	if err != nil {
		return fmt.Errorf("failed to create launcher backend: %w", err)
	}

	meta, err := cfg.Metadata.NewBackend(awsConfig)
	if err != nil {
		return fmt.Errorf("failed to create metadata backend: %w", err)
	}

	signer, err := cfg.Signing.NewBackend(awsConfig)
	if err != nil {
		return fmt.Errorf("failed to create signing backend: %w", err)
	}

	store, err := cfg.Storage.NewBackend(awsConfig)
	if err != nil {
		return fmt.Errorf("failed to open storage: %w", err)
	}

	tsClient := cfg.Tailscale.NewClient()

	logrus.Info("generating metadata documents")
	if err := generator.New(meta, signer).Serve(context.Background(), types.GenerateRequest{Issuer: gateway.BaseUrl}); err != nil {
		return fmt.Errorf("failed to generate metadata documents: %w", err)
	}

	var stopLauncher func()
	if cfg.Launcher.Backend == "local" {
		stopLauncher = startLauncher(bus, store, cfg.Tailscale.Tailnet)
	} else {
		stopLauncher = func() {}
	}

	srv, serverErrors := startGateway(tsClient, launch, meta, signer, store)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.WithError(err).Fatal("failed to start server")
		}

	case sig := <-shutdown:
		logrus.WithField("signal", sig.String()).Info("signal received, shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		stopLauncher()

		if err := srv.Shutdown(ctx); err != nil {
			logrus.WithError(err).Fatal("failed to shutdown server")
		}
	}

	logrus.Info("successfully shutdown, goodbye!")
	return nil
}
