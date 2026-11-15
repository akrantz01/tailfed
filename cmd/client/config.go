package main

var cfg Config

type Config struct {
	LogLevel string `koanf:"log-level" short:"l" help:"The minimum level to log at (choices: panic, fatal, error, warn, info, debug, trace)"`
	PidFile  string `koanf:"pid-file" help:"The path to write the daemon's PID"`
	Path     string `koanf:"path" short:"d" help:"The path to write the generated ID token to"`
	Profile  string `koanf:"profile" short:"p" help:"The profile to generate tokens for"`
	Url      string `koanf:"url" short:"u" help:"The URL of the Tailfed API"`
}

func (c *Config) SetDefaults() {
	c.LogLevel = "info"
	c.PidFile = "/run/tailfed/pid"
	c.Path = "/run/tailfed/token"
	c.Profile = "default"
}
