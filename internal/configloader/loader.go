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
	"github.com/spf13/pflag"
)

// RawConfig contains the raw configuration keys before structuring or validation.
type RawConfig struct {
	inner *koanf.Koanf
}

// Load retrieves configuration from at least the environment and command line arguments
func Load(opts ...Option) (*RawConfig, error) {
	options := &options{}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}

	k := koanf.New(".")

	secrets := newSecretLoader(options.awsConfig)
	envCleaner := newEnvVarCleaner(options.envPrefix, secrets)

	if len(options.file) != 0 {
		if err := k.Load(file.Provider(options.file), yaml.Parser()); err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to load from file %q: %w", options.file, err)
			}
		}
	}

	if err := k.Load(file.Provider(".env"), dotenv.ParserEnvWithValue(options.envPrefix, ".", envCleaner)); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load from dotenv: %w", err)
		}
	} else if err := secrets.Err(); err != nil {
		return nil, fmt.Errorf("failed to load secret from dotenv: %w", err)
	}

	if err := k.Load(env.ProviderWithValue(options.envPrefix, ".", envCleaner), nil); err != nil {
		return nil, fmt.Errorf("failed to load from env: %w", err)
	} else if err := secrets.Err(); err != nil {
		return nil, fmt.Errorf("failed to load secret from env: %w", err)
	}

	if options.flags != nil {
		if err := k.Load(posflag.ProviderWithFlag(options.flags, ".", k, func(f *pflag.Flag) (string, any) {
			// Only plain strings can reference secrets
			if f.Value.Type() == "string" {
				return secrets.Load(f.Name, f.Value.String())
			}

			return f.Name, posflag.FlagVal(options.flags, f)
		}), nil); err != nil {
			return nil, fmt.Errorf("failed to load from flags: %w", err)
		} else if err := secrets.Err(); err != nil {
			return nil, fmt.Errorf("failed to load secret from flags: %w", err)
		}
	}

	return &RawConfig{k}, nil
}

// LoadInto retrieves configuration from at least the environment and command line arguments and extracts it into a struct
func LoadInto[T any](dest *T, opts ...Option) error {
	raw, err := Load(opts...)
	if err != nil {
		return err
	}

	if err := raw.Structure(dest); err != nil {
		return fmt.Errorf("unable to unmarshal config: %w", err)
	}

	return nil
}

// Structure populates a struct with configuration values
func (r *RawConfig) Structure(dest any) error {
	return r.inner.Unmarshal("", dest)
}

func newEnvVarCleaner(prefix string, secrets *secretLoader) func(string, string) (string, any) {
	keyCleaner := func(s string) string {
		formatted := strings.ToLower(strings.TrimPrefix(s, prefix))
		nested := strings.ReplaceAll(formatted, "__", ".")
		return strings.ReplaceAll(nested, "_", "-")
	}

	return func(key, value string) (string, interface{}) {
		key = keyCleaner(key)
		return secrets.Load(key, value)
	}
}
