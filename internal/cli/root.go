package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const envPrefix = "TAILFED_"

type root struct {
	cmd  *cobra.Command
	exit func(int)

	LogLevel string `koanf:"log-level"`
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
		SilenceUsage:      true,
		SilenceErrors:     true,
		PersistentPreRunE: root.PersistentPreRun,
		RunE:              root.Run,
	}

	cmd.PersistentFlags().StringP("config", "c", "tailfed.yml", "The path to the configuration file")
	cmd.PersistentFlags().StringP("log-level", "l", "info", "The minimum level to log at (choices: panic, fatal, error, warn, info, debug, trace)")

	cmd.Flags().StringP("path", "p", "/run/tailfed/token", "The path to write the generated web identity token to")
	cmd.Flags().DurationP("frequency", "f", 1*time.Hour, "How often to refresh the token")
	cmd.Flags().StringP("url", "u", "", "The URL of the Tailfed API")

	cmd.AddCommand(root.NewRunCommand(), newVersion())

	root.cmd = cmd
	return root
}

// PersistentPreRun performs the common initializations for all commands
func (r *root) PersistentPreRun(*cobra.Command, []string) error {
	if err := r.loadConfig(); err != nil {
		return err
	}

	if err := logging.Initialize(r.LogLevel); err != nil {
		return err
	}

	return nil
}

func (r *root) loadConfig() error {
	k := koanf.New(".")
	flags := r.cmd.Flags()

	path, _ := flags.GetString("config")
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load config from file %q: %w", path, err)
	}

	if err := k.Load(env.Provider(envPrefix, ".", cleanEnvVarKey), nil); err != nil {
		return fmt.Errorf("failed to load config from env: %w", err)
	}

	if err := k.Load(posflag.Provider(flags, ".", k), nil); err != nil {
		return fmt.Errorf("failed to load config from cli: %w", err)
	}

	if err := k.Unmarshal("", r); err != nil {
		return fmt.Errorf("unable to unmarshal config: %w", err)
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

func cleanEnvVarKey(s string) string {
	formatted := strings.ToLower(strings.TrimPrefix(s, envPrefix))
	nested := strings.Replace(formatted, "__", ".", -1)
	return strings.Replace(nested, "_", "-", -1)
}
