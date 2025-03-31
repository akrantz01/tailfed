package main

import (
	"errors"
	"fmt"
	"slices"

	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/akrantz01/tailfed/internal/tailscale"
)

var (
	storageBackends = []string{"filesystem"}

	cfg config
)

type config struct {
	LogLevel string `koanf:"log-level"`
	Address  string `koanf:"address"`

	Storage   storageConfig   `koanf:"storage"`
	Tailscale tailscaleConfig `koanf:"tailscale"`
}

func (c *config) Validate() error {
	if !slices.Contains(storageBackends, c.Storage.Backend) {
		return fmt.Errorf("unknown storage backend %q", c.Storage.Backend)
	}

	if c.Storage.Backend == "filesystem" && len(c.Storage.Path) == 0 {
		return errors.New("missing path for filesystem backend")
	}

	if len(cfg.Tailscale.Tailnet) == 0 {
		return errors.New("a tailnet must be configured")
	}

	if (len(c.Tailscale.ApiKey) > 0) == c.Tailscale.OAuth.Enabled() {
		return errors.New("exactly one tailscale authentication method must be enabled")
	}

	return nil
}

type storageConfig struct {
	Backend string `koanf:"backend"`
	Path    string `koanf:"path"`
}

func (s *storageConfig) NewBackend() (storage.Backend, error) {
	switch s.Backend {
	case "filesystem":
		return storage.NewFilesystem(s.Path)
	default:
		return nil, errors.New("unknown storage backend")
	}
}

type tailscaleConfig struct {
	Tailnet string `koanf:"tailnet"`

	ApiKey string               `koanf:"api-key"`
	OAuth  tailscaleOAuthConfig `koanf:"oauth"`
}

func (t *tailscaleConfig) NewClient() *tailscale.API {
	return tailscale.NewAPI(t.Tailnet, t.Authentication())
}

func (t *tailscaleConfig) Authentication() tailscale.Authentication {
	if len(t.ApiKey) > 0 {
		return tailscale.ApiKey(t.ApiKey)
	}

	return tailscale.OAuth(t.OAuth.ClientId, t.OAuth.ClientSecret)
}

type tailscaleOAuthConfig struct {
	ClientId     string `koanf:"client-id"`
	ClientSecret string `koanf:"client-secret"`
}

func (o *tailscaleOAuthConfig) Enabled() bool {
	return len(o.ClientId) != 0 && len(o.ClientSecret) != 0
}
