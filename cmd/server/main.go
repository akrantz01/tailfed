package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akrantz01/tailfed/internal/configloader"
	"github.com/akrantz01/tailfed/internal/http"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	info := version.GetInfo()

	cmd := &cobra.Command{
		Use:           "tailfed-server",
		Version:       info.Version,
		Short:         "The tailfed OpenID Connect server",
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE:       preRun,
		RunE:          run,
	}
	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.Flags().StringP("log-level", "l", "info", "The minimum level to log at (choices: panic, fatal, error, warn, info, debug, trace)")
	cmd.Flags().StringP("address", "a", "127.0.0.1:8000", "The host and port to listen on")

	if err := cmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

}

func preRun(cmd *cobra.Command, _ []string) error {
	if err := configloader.LoadInto(&cfg, configloader.WithFlags(cmd.Flags()), configloader.WithEnvPrefix("TAILFED_SERVER_")); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if err := logging.Initialize(cfg.LogLevel); err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}

	return nil
}

func run(*cobra.Command, []string) error {
	srv := http.New()
	l, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", cfg.Address, err)
	}

	srvErrors := make(chan error, 1)
	go func() {
		logrus.WithField("address", l.Addr().String()).Info("server is ready")
		srvErrors <- srv.Serve(l)
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-srvErrors:
		return fmt.Errorf("http error: %w", err)
	case sig := <-shutdown:
		logrus.WithField("signal", sig.String()).Info("signal received, shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown server: %w", err)
		}
	}

	logrus.Info("successfully shutdown")
	logrus.Info("goodbye!")
	return nil
}
