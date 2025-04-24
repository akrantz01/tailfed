package cli

import (
	_ "embed"
	"fmt"
	"os"
	"text/template"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	//go:embed tailfed.yml.tpl
	configTemplateSrc string

	configTemplate = template.Must(template.New("tailfed.yml").Parse(configTemplateSrc))
)

var generateConfigCmd = &cobra.Command{
	Use:           "generate-config",
	Short:         "Generate a new configuration file",
	Long:          "Generates a new configuration file in the current directory using the provided backend API URL.",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ExactArgs(1),
	RunE:          generateConfig,
}

func generateConfig(cmd *cobra.Command, args []string) error {
	var path string
	flag := cmd.Flag("config")
	if flag.Changed {
		path = flag.Value.String()
	} else {
		path = flag.DefValue
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	ctx := map[string]any{
		"Url": args[0],
	}
	if err := configTemplate.Execute(file, ctx); err != nil {
		return fmt.Errorf("failed to generate config file: %w", err)
	}

	logrus.WithField("path", path).Info("generated config file")
	return nil
}
