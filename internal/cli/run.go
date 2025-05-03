package cli

import (
	"fmt"
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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

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

	apiClient, err := api.NewClient(http.DefaultClient, r.Url)
	if err != nil {
		return err
	}

	tsClient := tailscale.NewLocal(logrus.WithField("component", "tailscale"))
	refresh := refresher.New(apiClient, tsClient, r.Path)

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
