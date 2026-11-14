package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akrantz01/tailfed/internal/configloader"
	"github.com/akrantz01/tailfed/internal/http"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/akrantz01/tailfed/internal/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"tailscale.com/ipn"
)

func main() {
	cmd := &cobra.Command{
		Use:           "tailfed-server",
		Short:         "The tailfed OpenID Connect server",
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE:       preRun,
		RunE:          run,
	}
	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.Flags().StringP("config", "c", "tailfed-server.yml", "The path to the configuration file")
	cmd.Flags().Bool("json", false, "Format all output as JSON")
	version.AddFlag(cmd.Flags())
	configloader.BindFlags(cmd.Flags(), &Config{
		LogLevel: "info",
		Listener: listenerConfig{
			Address: "127.0.0.1:8000",
		},
		Signing: signingConfig{
			Backend: "memory",
		},
		Tailscale: tailscaleConfig{
			Connection:      tailscale.ConnectionLocal,
			ControlPlaneUrl: ipn.DefaultControlURL,
			Authentication:  tailscale.AuthenticationMethodKey,
		},
		Token: tokenConfig{
			Validity: 1 * time.Hour,
		},
	})

	if err := cmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

}

func preRun(cmd *cobra.Command, _ []string) error {
	flags := cmd.Flags()
	path, _ := flags.GetString("config")
	if err := configloader.LoadInto(&cfg,
		configloader.WithFlags(flags),
		configloader.WithEnvPrefix("TAILFED_SERVER_"),
		configloader.IncludeConfigFile(path),
	); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	json, _ := flags.GetBool("json")
	if err := logging.Initialize(cfg.LogLevel, json); err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}

	fmt.Printf("%+v\n", cfg)

	return nil
}

func run(cmd *cobra.Command, _ []string) error {
	if version.Display(cmd) {
		return nil
	}

	awsConfig, err := cfg.LoadAWSConfig(context.Background())
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	signer, err := cfg.Signing.NewBackend(awsConfig)
	if err != nil {
		return fmt.Errorf("failed to create signing backend: %w", err)
	}

	ts, err := cfg.Tailscale.Connect(context.Background())
	if err != nil {
		return fmt.Errorf("failed to connect to tailscale: %w", err)
	}

	srv := http.New(signer, ts)
	l, err := ts.Listen(context.Background(), cfg.Listener.Address, cfg.Listener.Service)
	if err != nil {
		return fmt.Errorf("failed to get listener: %w", err)
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
