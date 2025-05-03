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

type generateConfig struct {
	Path string `koanf:"config"`
}

func newGenerateConfig() *cobra.Command {
	gc := &generateConfig{}
	return &cobra.Command{
		Use:           "generate-config",
		Short:         "Generate a new configuration file",
		Long:          "Generates a new configuration file in the current directory using the provided backend API URL.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		PreRunE:       structureConfigInto(gc),
		RunE:          gc.Run,
	}
}

func (gc *generateConfig) Run(_ *cobra.Command, args []string) error {
	file, err := os.Create(gc.Path)
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

	logrus.WithField("path", gc.Path).Info("generated config file")
	return nil
}
