package main

var cfg Config

type Config struct {
	LogLevel string `koanf:"log-level" short:"l" help:"The minimum level to log at (choices: panic, fatal, error, warn, info, debug, trace)"`
	PidFile  string `koanf:"pid-file" help:"The path to write the daemon's PID"`
	Path     string `koanf:"path" short:"p" help:"The path to write the generated ID token to"`
	Url      string `koanf:"url" short:"u" help:"The URL of the Tailfed API"`
}
