package main

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/akrantz01/tailfed/internal/api"
	"github.com/akrantz01/tailfed/internal/configloader"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/refresher"
	"github.com/akrantz01/tailfed/internal/scheduler"
	"github.com/akrantz01/tailfed/internal/systemd"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/akrantz01/tailfed/internal/version"
	hashiversion "github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var remoteVersionCompat = hashiversion.MustConstraints(hashiversion.NewConstraint(">= 1.0, < 2.0"))

func main() {
	cmd := &cobra.Command{
		Use:   "tailfed-client",
		Short: "A daemon for refreshing AWS identity tokens via Tailfed.",
		Long: `
A daemon for refreshing AWS web identity federation tokens issued by Tailfed. Tailfed uses your
Tailscale network to prove a host's identity, allowing it to retrieve temporary AWS credentials.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE:       preRun,
		RunE:          run,
	}
	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.Flags().StringP("config", "c", "tailfed.yml", "The path to the configuration file")
	cmd.Flags().Bool("json", false, "Format all output as JSON")
	version.AddFlag(cmd.Flags())
	configloader.BindFlags(cmd.Flags(), &Config{
		LogLevel: "info",
		PidFile:  "/run/tailfed/pid",
		Path:     "/run/tailfed/token",
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
		configloader.WithEnvPrefix("TAILFED_"),
		configloader.IncludeConfigFile(path),
	); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
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

	if err := writePidFile(cfg.PidFile); err != nil {
		logrus.WithField("path", cfg.PidFile).WithError(err).Error("failed to write pid file")
	} else {
		defer removePidFile(cfg.PidFile)
	}

	return daemon(cmd.Context())
}

func daemon(ctx context.Context) error {
	systemd.EnableWatchdog()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	client := &http.Client{
		Transport: newAddHeaderTransport(nil, http.Header{
			"User-Agent": []string{fmt.Sprintf("tailfed-client/%s", version.GetInfo().Version)},
		}),
	}

	apiClient, err := api.NewClient(client, cfg.Url)
	if err != nil {
		return err
	}

	tsClient, _ := tailscale.NewLocal(context.Background())
	refresh := refresher.New(apiClient, tsClient, cfg.Path)

	if err := verifyApiVersion(ctx, apiClient); err != nil {
		return err
	}

	config, err := apiClient.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get daemon config from api: %w", err)
	}
	logrus.
		WithFields(map[string]any{
			"frequency": config.Frequency.String(),
		}).
		Info("got daemon config")

	sched := scheduler.NewScheduler(ctx, time.Duration(config.Frequency), refresh.Job)

	sched.Start()

	logrus.Info("daemon started")
	systemd.Ready()

signals:
	for {
		switch <-sigs {
		case syscall.SIGHUP:
			systemd.Reloading()

			sched.RunNow()
			logrus.Info("reload received, refreshing token now")

			systemd.Ready()

		case syscall.SIGINT, syscall.SIGTERM:
			break signals
		}
	}

	logrus.Info("signal received, shutting down...")
	systemd.Stopping()

	sched.Stop()
	refresh.ShutdownInFlight()

	return nil
}

// writePidFile writes the process' PID to the given file
func writePidFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	if _, err := file.WriteString(strconv.Itoa(os.Getpid())); err != nil {
		return fmt.Errorf("failed to write pid: %w", err)
	}

	return nil
}

// removePidFile removes the PID file, logging any failures
func removePidFile(path string) {
	if err := os.Remove(path); err != nil {
		logrus.WithField("path", path).WithError(err).Error("failed to remove pid file")
	}
}

func verifyApiVersion(ctx context.Context, client *api.Client) error {
	remote, err := client.GetVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get api version: %w", err)
	}
	logrus.
		WithFields(map[string]any{
			"version":  remote.Version,
			"revision": remote.Commit,
		}).
		Info("got remote version info")

	remoteVersion, err := hashiversion.NewSemver(remote.Version)
	if err != nil {
		return fmt.Errorf("failed to parse api version %q: %w", remote.Version, err)
	}

	if remoteVersionCompat.Check(remoteVersion) {
		return nil
	} else {
		return fmt.Errorf("remote api version %q is incompatible with daemon version %q", remote.Version, version.GetInfo().Version)
	}
}

type addHeaderTransport struct {
	inner   http.RoundTripper
	headers http.Header
}

var _ http.RoundTripper = (*addHeaderTransport)(nil)

func newAddHeaderTransport(transport http.RoundTripper, headers http.Header) http.RoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}

	return &addHeaderTransport{transport, headers}
}

func (aht *addHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	maps.Copy(req.Header, aht.headers)
	return aht.inner.RoundTrip(req)
}
