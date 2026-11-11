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
	"tailscale.com/ipn"
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
	cmd.Flags().StringP("listener.address", "", "127.0.0.1:8000", "The host and port to listen on")
	cmd.Flags().StringP("listener.service", "", "", "The Tailscale service to listen on")
	cmd.Flags().StringP("tailscale.connection", "", string(TailscaleConnectionLocal), "How to connect to Tailscale (options: local, in-process)")
	cmd.Flags().StringP("tailscale.control-plane-url", "", ipn.DefaultControlURL, "The Tailscale/Headscale control plane to connect to (in-process only)")
	cmd.Flags().StringP("tailscale.authentication", "", string(TailscaleAuthMethodAuthKey), "How to authenticate with the control plane (in-process only) (options: authkey, oauth, workload)")
	cmd.Flags().StringP("tailscale.auth-key", "", "", "The Tailscale authkey used to join the tailnet (in-process only)")
	cmd.Flags().StringP("tailscale.client-id", "", "", "The Tailscale client ID to generate authkey(s) using OAuth or workload identity federation (in-process only)")
	cmd.Flags().StringP("tailscale.client-secret", "", "", "The Tailscale OAuth client secret to generate authkey(s) (in-process only)")
	cmd.Flags().StringP("tailscale.id-token", "", "", "The identity provider's ID token for workload identity federation-based authkey generation (in-process only)")
	cmd.Flags().StringP("tailscale.audience", "", "", "The audience used to automatically fetch an ID token (in-process only)")

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

	fmt.Printf("%+v\n", cfg)

	return nil
}

func run(*cobra.Command, []string) error {
	srv := http.New()
	l, err := net.Listen("tcp", cfg.Listener.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", cfg.Listener.Address, err)
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
