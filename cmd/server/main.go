package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
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
	"golang.org/x/sync/errgroup"
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

	var listeners []listener

	privateSrv := http.NewPrivate(issuer, profiles, signer, ts)

	if cfg.Listener.HTTP {
		httpLn, err := ts.ListenPrivate(context.Background(), "127.0.0.1:80")
		if err != nil {
			return fmt.Errorf("failed to listen start http listener: %w", err)
		}

		httpSrv := privateSrv
		if httpsUrl := cfg.Listener.HttpsUrl(name); httpsUrl != "" {
			httpSrv = http.NewRedirect(httpsUrl)
		}
		listeners = append(listeners, listener{"http", httpSrv, httpLn})
	}

	if cfg.Listener.HTTPS {
		httpsLn, err := ts.ListenPrivate(context.Background(), "127.0.0.1:443")
		if err != nil {
			return fmt.Errorf("failed to start https listener: %w", err)
		}

		tlsLn := tls.NewListener(httpsLn, &tls.Config{
			GetCertificate: ts.GetCertificate,
		})
		listeners = append(listeners, listener{"https", privateSrv, tlsLn})
	}

	if cfg.Listener.Funnel {
		publicSrv := http.NewPublic(issuer, profiles, signer)

		publicLn, err := ts.ListenFunnel(context.Background())
		if err != nil {
			return fmt.Errorf("failed to listen on funnel: %w", err)
		}

		listeners = append(listeners, listener{"funnel", publicSrv, publicLn})
	}

	serveCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := serve(serveCtx, listeners); err != nil {
		return err
	}

	logrus.Info("successfully shutdown")
	logrus.Info("goodbye!")
	return nil
}

type listener struct {
	name string
	srv  *http.Server
	ln   net.Listener
}

func serve(ctx context.Context, listeners []listener) error {
	errGrp, ctx := errgroup.WithContext(ctx)

	for _, l := range listeners {
		errGrp.Go(func() error {
			logrus.WithField("listener.name", l.name).WithField("listener.address", l.ln.Addr().String()).Info("listener is ready")
			if err := l.srv.Serve(l.ln); !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("%s listener error: %w", l.name, err)
			}
			return nil
		})
	}

	errGrp.Go(func() error {
		<-ctx.Done()

		logrus.Info("signal received, shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var errs []error
		for _, l := range listeners {
			if err := l.srv.Shutdown(shutdownCtx); err != nil {
				errs = append(errs, fmt.Errorf("failed to shutdown %s listener: %w", l.name, err))
			}
		}

		return errors.Join(errs...)
	})

	return errGrp.Wait()
}
