package configloader

type options struct {
	file      string
	envPrefix string
}

type Option func(opts *options)

// WithEnvPrefix adds a prefix to environment variable options
func WithEnvPrefix(prefix string) Option {
	return func(opts *options) {
		opts.envPrefix = prefix
	}
}

// IncludeConfigFile loads configuration from a file as the initial source
func IncludeConfigFile(name string) Option {
	return func(opts *options) {
		opts.file = name
	}
}
