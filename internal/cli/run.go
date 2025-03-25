package cli

import (
	"context"
	"fmt"
	"net/url"
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
		PreRunE: r.PreRun,
		RunE:    r.Run,
	}
}

func (r *run) PreRun(*cobra.Command, []string) error {
	if len(r.url) == 0 {
		return fmt.Errorf("missing required option 'url'")
	}

	u, err := url.Parse(r.url)
	if err != nil {
		return fmt.Errorf("invalid backend url: %w", err)
	}

	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("invalid backend url: scheme must be 'http' or 'https', got '%s'", u.Scheme)
	}

	return nil
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
