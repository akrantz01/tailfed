package configloader

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/pflag"
)

type options struct {
	flags     *pflag.FlagSet
	file      string
	envPrefix string
	awsConfig *aws.Config
}

type Option func(opts *options)

// WithFlags loads configuration from command line arguments
func WithFlags(flags *pflag.FlagSet) Option {
	return func(opts *options) {
		opts.flags = flags
	}
}

// WithEnvPrefix adds a prefix to environment variable options
func WithEnvPrefix(prefix string) Option {
	return func(opts *options) {
		opts.envPrefix = prefix
	}
}

// WithSecrets allows loading secrets from AWS secrets manager and SSM parameter store
func WithSecrets(config aws.Config) Option {
	return func(opts *options) {
		opts.awsConfig = &config
	}
}

// IncludeConfigFile loads configuration from a file as the initial source
func IncludeConfigFile(name string) Option {
	return func(opts *options) {
		opts.file = name
	}
}
