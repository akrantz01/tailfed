package cli

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/akrantz01/tailfed/internal/configloader"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type root struct {
	cmd  *cobra.Command
	exit func(int)

	wrotePid bool

	LogLevel string `koanf:"log-level"`
	PidFile  string `koanf:"pid-file"`
	*run     `koanf:",squash"`
}

var _ Executable = (*root)(nil)

// NewRoot creates the root command for the client
func NewRoot(exit func(int)) Executable {
	root := &root{
		exit: exit,
		run:  &run{},
	}

	cmd := &cobra.Command{
		Use:   "tailfed-client",
		Short: "A daemon for refreshing AWS identity tokens via Tailfed.",
		Long: `
A daemon for refreshing AWS web identity federation tokens issued by Tailfed. Tailfed uses your
Tailscale network to prove a host's identity, allowing it to retrieve temporary AWS credentials.`,
		SilenceUsage:       true,
		SilenceErrors:      true,
		PersistentPreRunE:  root.PersistentPreRun,
		RunE:               root.Run,
		PersistentPostRunE: root.PersistentPostRun,
	}

	cmd.PersistentFlags().StringP("config", "c", "tailfed.yml", "The path to the configuration file")
	cmd.PersistentFlags().StringP("log-level", "l", "info", "The minimum level to log at (choices: panic, fatal, error, warn, info, debug, trace)")
	cmd.PersistentFlags().String("pid-file", "/run/tailfed/pid", "The path to read/write the daemon's PID")

	cmd.Flags().StringP("path", "p", "/run/tailfed/token", "The path to write the generated web identity token to")
	cmd.Flags().DurationP("frequency", "f", 1*time.Hour, "How often to refresh the token")
	cmd.Flags().StringP("url", "u", "", "The URL of the Tailfed API")

	cmd.AddCommand(root.NewRunCommand(), newGenerateConfig(), newRefresh(), newVersion())

	root.cmd = cmd
	return root
}

// PersistentPreRun performs the common initializations for all commands
func (r *root) PersistentPreRun(cmd *cobra.Command, _ []string) error {
	flags := cmd.Flags()
	path, _ := flags.GetString("config")
	config, err := configloader.Load(
		configloader.WithFlags(flags),
		configloader.WithEnvPrefix("TAILFED_"),
		configloader.IncludeConfigFile(path),
	)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := config.Structure(r); err != nil {
		return fmt.Errorf("failed to structure root config: %w", err)
	}

	ctx := config.InContext(cmd.Context())
	cmd.SetContext(ctx)

	if err := logging.Initialize(r.LogLevel); err != nil {
		return err
	}

	if err := r.writePidFile(cmd.Name()); err != nil {
		logrus.WithField("path", r.PidFile).WithError(err).Error("failed to write pid file")
	}

	return nil
}

// Execute runs the command with the provided arguments/options
func (r *root) Execute(args []string) {
	r.cmd.SetArgs(args)

	err := r.cmd.Execute()
	if err != nil {
		logrus.Error(err.Error())
		r.exit(1)
	}
}

// PersistentPostRun performs common cleanup for all commands
func (r *root) PersistentPostRun(*cobra.Command, []string) error {
	if r.wrotePid {
		if err := os.Remove(r.PidFile); err != nil {
			logrus.WithField("path", r.PidFile).WithError(err).Error("failed to remove pid file")
		}
	}

	return nil
}

// writePidFile writes the process' PID to the given file
func (r *root) writePidFile(cmd string) error {
	if cmd != "tailfed-client" && cmd != "run" {
		return nil
	}

	pid := os.Getpid()

	file, err := os.Create(r.PidFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(strconv.Itoa(pid)); err != nil {
		return fmt.Errorf("failed to write pid: %w", err)
	}

	r.wrotePid = true
	return nil
}
