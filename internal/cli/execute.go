package cli

import (
	"errors"
	"fmt"

	"github.com/akrantz01/tailfed/internal/configloader"
	"github.com/spf13/cobra"
)

// Executable is a command that can be run with some pre-provided arguments/options
type Executable interface {
	Execute(args []string)
}

// Execute runs the root command
func Execute(exit func(int), args []string) {
	NewRoot(exit).Execute(args)
}

// structureConfigInto generates a command pre-run step to structure raw config into command-specific config. Assumes
// the raw config was parsed by the root command and exists in the command context.
func structureConfigInto[T any](dest *T) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		config := configloader.FromContext(cmd.Context())
		if config == nil {
			return errors.New("config not present in command context")
		}

		if err := config.Structure(dest); err != nil {
			return fmt.Errorf("failed to structure command config: %w", err)
		}

		return nil
	}
}
