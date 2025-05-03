package cli

import (
	"github.com/akrantz01/tailfed/internal/version"
	"github.com/spf13/cobra"
)

func newVersion() *cobra.Command {
	var json bool

	cmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Prints the version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := version.GetInfo()
			info.Name = cmd.Root().Name()
			info.Description = cmd.Root().Short

			cmd.SetOut(cmd.OutOrStdout())

			if json {
				cmd.Println(info.JSON())
			} else {
				cmd.Println(info.String())
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&json, "json", false, "Print JSON instead of text")

	return cmd
}
