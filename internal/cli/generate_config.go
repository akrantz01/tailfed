package cli

import (
	_ "embed"
	"errors"
	"fmt"
	"net/url"
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
	apiUrl, err := gc.parseUrl(args[0])
	if err != nil {
		return fmt.Errorf("invalid api url %q: %w", args[0], err)
	}

	// TODO: test url is valid

	if err := gc.writeConfig(apiUrl); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	logrus.WithField("path", gc.Path).Info("generated config file")
	return nil
}

func (gc *generateConfig) parseUrl(raw string) (*url.URL, error) {
	apiUrl, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid format: %w", err)
	}

	if apiUrl.Scheme != "http" && apiUrl.Scheme != "https" {
		return nil, errors.New("must be either 'http' or 'https'")
	}

	if len(apiUrl.Hostname()) == 0 {
		return nil, errors.New("missing hostname")
	}

	return apiUrl, nil
}

func (gc *generateConfig) writeConfig(apiUrl *url.URL) error {
	file, err := os.Create(gc.Path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	ctx := map[string]any{
		"Url": apiUrl.String(),
	}
	if err := configTemplate.Execute(file, ctx); err != nil {
		return fmt.Errorf("failed to generate config file: %w", err)
	}

	return nil
}
