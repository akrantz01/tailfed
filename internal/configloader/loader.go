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
		if err := k.Load(posflag.ProviderWithValue(options.flags, ".", k, secrets.Load), nil); err != nil {
			return nil, fmt.Errorf("failed to load from flags: %w", err)
		} else if err := secrets.Err(); err != nil {
			return nil, fmt.Errorf("failed to load secret from flags: %w", err)
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

func newEnvVarCleaner(prefix string, secrets *secretLoader) func(string, string) (string, any) {
	keyCleaner := func(s string) string {
		formatted := strings.ToLower(strings.TrimPrefix(s, prefix))
		nested := strings.Replace(formatted, "__", ".", -1)
		return strings.Replace(nested, "_", "-", -1)
	}

	return func(key, value string) (string, interface{}) {
		key = keyCleaner(key)
		return secrets.Load(key, value)
	}
}
