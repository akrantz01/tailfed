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
	"github.com/akrantz01/tailfed/internal/oidc"
	"github.com/akrantz01/tailfed/internal/version"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var awsConfig aws.Config

func main() {
	cmd := &cobra.Command{
		Use:           "tailfed-server",
		Short:         "The tailfed OpenID Connect server",
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE:       preRun,
		RunE:          run,
	}

	cmd.Flags().StringP("config", "c", "tailfed-server.yml", "The path to the configuration file")
	cmd.Flags().Bool("json", false, "Format all output as JSON")
	version.AddFlag(cmd.Flags())
	configloader.BindFlags[Config](cmd.Flags())

	if err := cmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

}

func preRun(cmd *cobra.Command, _ []string) error {
	var err error
	awsConfig, err = awsconfig.LoadDefaultConfig(cmd.Context())
	if err != nil {
		logrus.WithError(err).Error("failed to load aws config")
	}

	flags := cmd.Flags()
	path, _ := flags.GetString("config")
	if err := configloader.LoadInto(&cfg,
		configloader.WithFlags(flags),
		configloader.WithEnvPrefix("TAILFED_SERVER_"),
		configloader.IncludeConfigFile(path),
		configloader.WithSecrets(awsConfig),
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

	return nil
}

func run(cmd *cobra.Command, _ []string) error {
	if version.Display(cmd) {
		return nil
	}

	signer, err := cfg.Signing.NewBackend(awsConfig)
	if err != nil {
		return fmt.Errorf("failed to create signing backend: %w", err)
	}

	ts, err := cfg.Tailscale.Connect(context.Background())
	if err != nil {
		return fmt.Errorf("failed to connect to tailscale: %w", err)
	}

	profiles := oidc.NewProfiles(cfg.Token.DefaultProfile, cfg.Token.Profiles)

	name, err := ts.DNSName(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get dns name: %w", err)
	}
	issuer := cfg.Listener.Issuer(name)
	logrus.WithField("oidc.issuer", issuer).Info("selected issuer from magic dns")

	privateSrv := http.NewPrivate(issuer, profiles, signer, ts)
	privateLn, err := ts.ListenPrivate(context.Background(), cfg.Listener.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on %q: %w", cfg.Listener.Address, err)
	}

	privateSrvErr := make(chan error, 1)
	go func() {
		logrus.WithField("address", privateLn.Addr().String()).Info("private server is ready")
		privateSrvErr <- privateSrv.Serve(privateLn)
	}()

	var publicSrv *http.Server
	publicSrvErr := make(chan error, 1)
	if cfg.Listener.Funnel {
		publicSrv = http.NewPublic(issuer, profiles, signer)
		publicLn, err := ts.ListenFunnel(context.Background())
		if err != nil {
			return fmt.Errorf("failed to listen on funnel: %w", err)
		}

		go func() {
			logrus.WithField("issuer", issuer).Info("public server is ready")
			publicSrvErr <- publicSrv.Serve(publicLn)
		}()
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-privateSrvErr:
		return fmt.Errorf("private http server error: %w", err)
	case err := <-publicSrvErr:
		return fmt.Errorf("public http server error: %w", err)
	case sig := <-shutdown:
		logrus.WithField("signal", sig.String()).Info("signal received, shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := privateSrv.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown private server: %w", err)
		}
		if publicSrv != nil {
			if err := publicSrv.Shutdown(ctx); err != nil {
				return fmt.Errorf("failed to shutdown public server: %w", err)
			}
		}
	}

	logrus.Info("successfully shutdown")
	logrus.Info("goodbye!")
	return nil
}
