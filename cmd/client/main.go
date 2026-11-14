package main

import (
	"context"
	"fmt"
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
	"github.com/akrantz01/tailfed/internal/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

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
	configloader.BindFlags[Config](cmd.Flags())

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

	client, err := api.NewClient(nil, cfg.Url)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	refresh := refresher.New(client, cfg.Path, cfg.Profile)

	sched := scheduler.NewScheduler(ctx, 1*time.Hour, refresh.Job)
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
