package main

import (
	"fmt"
	"net"
)

var cfg Config

type Config struct {
	LogLevel string `koanf:"log-level"`

	Address string `koanf:"address"`
}

func (c *Config) Validate() error {
	if _, _, err := net.SplitHostPort(c.Address); err != nil {
		return fmt.Errorf("invalid listen address: %w", err)
	}

	return nil
}
