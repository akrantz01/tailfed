package configloader

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
)

// Load retrieves configuration from at least the environment and command line arguments
func Load(opts ...Option) (*koanf.Koanf, error) {
	options := &options{}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}

	k := koanf.New(".")

	if len(options.file) != 0 {
		if err := k.Load(file.Provider(options.file), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("failed to load from file %q: %w", options.file, err)
		}
	}

	if err := k.Load(file.Provider(".env"), dotenv.ParserEnv(options.envPrefix, ".", cleanEnvVarKey(options.envPrefix))); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load from dotenv: %w", err)
		}
	}

	if err := k.Load(env.Provider(options.envPrefix, ".", cleanEnvVarKey(options.envPrefix)), nil); err != nil {
		return nil, fmt.Errorf("failed to load from env: %w", err)
	}

	if options.flags != nil {
		if err := k.Load(posflag.Provider(options.flags, ".", k), nil); err != nil {
			return nil, fmt.Errorf("failed to load from flags: %w", err)
		}
	}

	return k, nil
}

// LoadInto retrieves configuration from at least the environment and command line arguments and extracts it into a struct
func LoadInto[T any](dest *T, opts ...Option) error {
	k, err := Load(opts...)
	if err != nil {
		return err
	}

	if err := k.Unmarshal("", dest); err != nil {
		return fmt.Errorf("unable to unmarshal config: %w", err)
	}

	return nil
}

func cleanEnvVarKey(prefix string) func(string) string {
	return func(s string) string {
		formatted := strings.ToLower(strings.TrimPrefix(s, prefix))
		nested := strings.Replace(formatted, "__", ".", -1)
		return strings.Replace(nested, "_", "-", -1)
	}
}
