package configloader

import (
	"context"
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

// RawConfig contains the raw configuration keys before structuring or validation.
type RawConfig struct {
	inner *koanf.Koanf
}

type rawConfigKey struct{}

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
		if err := k.Load(posflag.ProviderWithValue(options.flags, ".", k, secrets.Load), nil); err != nil {
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

// FromContext retrieves a RawConfig instance from a context
func FromContext(ctx context.Context) *RawConfig {
	value := ctx.Value(rawConfigKey{})
	if value == nil {
		return nil
	}
	if r, ok := value.(*RawConfig); ok {
		return r
	} else {
		panic("value at RawConfig key is not a *RawConfig")
	}
}

// Structure populates a struct with configuration values
func (r *RawConfig) Structure(dest any) error {
	return r.inner.Unmarshal("", dest)
}

// InContext adds the RawConfig instance to a context
func (r *RawConfig) InContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, rawConfigKey{}, r)
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
