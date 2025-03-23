package cli

import (
	"github.com/spf13/cobra"
	"sigs.k8s.io/release-utils/version"
)

func newVersion() *cobra.Command {
	cmd := version.WithFont("basic")
	cmd.Aliases = []string{"v"}
	return cmd
}
