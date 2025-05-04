package cli

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akrantz01/tailfed/internal/api"
	"github.com/akrantz01/tailfed/internal/refresher"
	"github.com/akrantz01/tailfed/internal/scheduler"
	"github.com/akrantz01/tailfed/internal/systemd"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var remoteVersionCompat = version.MustConstraints(version.NewConstraint(">= 1.0, < 2.0"))

type run struct {
	Path string `koanf:"path"`
	Url  string `koanf:"url"`
}

func (r *run) NewRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "run",
		Aliases: []string{"r", "start", "s"},
		Short:   "Start the daemon (default)",
		Long:    "Starts the daemon using the configuration from the flags.",
		RunE:    r.Run,
	}
}

func (r *run) Run(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	systemd.EnableWatchdog()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	client := &http.Client{
		Transport: newAddHeaderTransport(nil, http.Header{
			"User-Agent": []string{fmt.Sprintf("tailfed-client/%s", cmd.Root().Version)},
		}),
	}

	apiClient, err := api.NewClient(client, r.Url)
	if err != nil {
		return err
	}

	tsClient := tailscale.NewLocal(logrus.WithField("component", "tailscale"))
	refresh := refresher.New(apiClient, tsClient, r.Path)

	if err := r.verifyApiVersion(ctx, cmd, apiClient); err != nil {
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

func (r *run) verifyApiVersion(ctx context.Context, cmd *cobra.Command, client *api.Client) error {
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

	remoteVersion, err := version.NewSemver(remote.Version)
	if err != nil {
		return fmt.Errorf("failed to parse api version %q: %w", remote.Version, err)
	}

	if remoteVersionCompat.Check(remoteVersion) {
		return nil
	} else {
		return fmt.Errorf("remote api version %q is incompatible with daemon version %q", remote.Version, cmd.Root().Version)
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
