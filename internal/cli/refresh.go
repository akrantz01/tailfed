package cli

import (
	"time"

	"github.com/spf13/cobra"
)

type refresh struct {
	PidFile string        `koanf:"pid-file"`
	Token   string        `koanf:"path"`
	Wait    bool          `koanf:"wait"`
	Timeout time.Duration `koanf:"timeout"`
}

func newRefresh() *cobra.Command {
	r := &refresh{}
	cmd := &cobra.Command{
		Use:           "refresh",
		Short:         "Force the daemon to refresh the token",
		Long:          "Forces the currently running daemon to request a new web identity token.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		PreRunE:       structureConfigInto(r),
		RunE:          r.Run,
	}

	cmd.Flags().BoolP("wait", "w", false, "Wait for the token to be refreshed before exiting")
	cmd.Flags().DurationP("timeout", "t", 30*time.Second, "How long to wait for the token to be refreshed")

	return cmd
}
