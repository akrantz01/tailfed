package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akrantz01/tailfed/internal/scheduler"
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

	sched := scheduler.NewScheduler(cmd.Context(), r.frequency, func(ctx context.Context) error {
		logrus.Info("job run")
		return nil
	})

	sched.Start()

	logrus.Info("daemon started")
	<-sigs
	logrus.Info("signal received, shutting down...")

	sched.Stop()

	return nil
}
