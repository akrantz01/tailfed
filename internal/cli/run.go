package cli

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akrantz01/tailfed/internal/api"
	"github.com/akrantz01/tailfed/internal/refresher"
	"github.com/akrantz01/tailfed/internal/scheduler"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type run struct {
	path      string
	frequency time.Duration
	url       string
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
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	apiClient, err := api.NewClient(r.url)
	if err != nil {
		return err
	}

	tsClient := tailscale.NewClient()
	refresh := refresher.New(apiClient, tsClient, r.path)

	sched := scheduler.NewScheduler(cmd.Context(), r.frequency, refresh.Job)

	sched.Start()

	logrus.Info("daemon started")
	<-sigs
	logrus.Info("signal received, shutting down...")

	sched.Stop()
	refresh.ShutdownInFlight()

	return nil
}
