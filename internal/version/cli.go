package version

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// AddFlag registers a --version/-v flag
func AddFlag(flags *pflag.FlagSet) {
	flags.BoolP("version", "v", false, "Display the current version")
}

// Display displays the version if the --version/-v flag was set, respecting a --json flag (if it exists)
func Display(cmd *cobra.Command) bool {
	flags := cmd.Flags()
	if version, _ := flags.GetBool("version"); !version {
		return false
	}

	json := jsonFlag(flags)

	info := GetInfo()
	info.Name = cmd.Root().Name()
	info.Description = cmd.Root().Short

	cmd.SetOut(cmd.OutOrStdout())

	if json {
		cmd.Println(info.JSON())
	} else {
		cmd.Println(info.String())
	}

	return true
}

func jsonFlag(flags *pflag.FlagSet) bool {
	if json, err := flags.GetBool("json"); err != nil {
		return false
	} else {
		return json
	}
}
