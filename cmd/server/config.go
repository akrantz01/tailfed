package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"

	"github.com/akrantz01/tailfed/internal/oidc"
	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/aws/aws-sdk-go-v2/aws"
	"tailscale.com/ipn"
)

var cfg Config

type Config struct {
	LogLevel string `koanf:"log-level" short:"l" help:"The minimum level to log at"`

	Listener  listenerConfig  `koanf:"listener"`
	Signing   signingConfig   `koanf:"signing"`
	Tailscale tailscaleConfig `koanf:"tailscale"`
	Token     tokenConfig     `koanf:"token"`
}

func (c *Config) SetDefaults() {
	c.LogLevel = "info"
}

func (c *Config) Validate() error {
	if err := c.Listener.Validate(); err != nil {
		return fmt.Errorf("invalid listener configuration: %w", err)
	}
	if err := c.Signing.Validate(); err != nil {
		return fmt.Errorf("invalid signing configuration: %w", err)
	}
	if err := c.Tailscale.Validate(); err != nil {
		return fmt.Errorf("invalid tailscale configuration: %w", err)
	}
	if err := c.Token.Validate(); err != nil {
		return fmt.Errorf("invalid token configuration: %w", err)
	}

	return nil
}

type listenerConfig struct {
	HTTP   bool `koanf:"http" help:"Enable serving over HTTP"`
	HTTPS  bool `koanf:"https" help:"Enable serving over HTTPS"`
	Funnel bool `koanf:"funnel" help:"Expose the token verification routes publicly via Tailscale Funnel"`
}

func (c *listenerConfig) SetDefaults() {
	c.HTTP = true
}

func (c *listenerConfig) Validate() error {
	if !c.HTTP && !c.HTTPS {
		return errors.New("either a http or https listener is required")
	}

	return nil
}

// Issuer builds the token issuer used in the discovery document and embedded in tokens
func (c *listenerConfig) Issuer(dnsName string) string {
	if c.Funnel || c.HTTPS {
		return "https://" + dnsName
	} else {
		return "http://" + dnsName
	}
}

// HttpsUrl is the base URL of the private https listener, or empty when it is disabled
func (c *listenerConfig) HttpsUrl(dnsName string) string {
	if !c.HTTPS {
		return ""
	}
	return "https://" + dnsName
}

type signingConfig struct {
	Backend signing.BackendId   `koanf:"backend" help:"How to sign ID tokens"`
	AwsKms  signingAwsKmsConfig `koanf:"aws-kms"`
	File    signingFileConfig   `koanf:"file"`
	Memory  signingMemoryConfig `koanf:"memory"`
}

func (c *signingConfig) SetDefaults() {
	c.Backend = "memory"
	c.Memory.RS256 = true
	c.Memory.ES256 = true
}

func (c *signingConfig) Validate() error {
	switch c.Backend {
	case signing.BackendMemory:
		return c.Memory.Validate()

	case signing.BackendFile:
		return c.File.Validate()

	case signing.BackendAWSKMS:
		return c.AwsKms.Validate()

	default:
		return fmt.Errorf("unknown signing backend %q (options: %s)", c.Backend, signing.BackendIds)
	}
}

func (c *signingConfig) NewBackend(awsConfig aws.Config) (signing.Backend, error) {
	switch c.Backend {
	case signing.BackendMemory:
		return signing.NewInMemory(c.Memory.Algorithms())

	case signing.BackendFile:
		var files []signing.File
		for _, file := range c.File.Paths {
			files = append(files, signing.File{
				Path:     file.Path,
				Generate: file.Generate,
			})
		}
		return signing.NewFile(files)

	case signing.BackendAWSKMS:
		return signing.NewAWSKMS(awsConfig, c.AwsKms.Keys)

	default:
		panic("unreachable")
	}
}

type signingAwsKmsConfig struct {
	Keys []string `koanf:"keys" help:"The AWS KMS key ARN to use"`
}

func (c *signingAwsKmsConfig) Validate() error {
	if len(c.Keys) == 0 {
		return errors.New("at least one KMS key ARN/alias is required")
	}
	return nil
}

type signingFileConfig struct {
	Paths []signingFileKeyConfig `koanf:"paths"`
}

func (c *signingFileConfig) Validate() error {
	if len(c.Paths) == 0 {
		return errors.New("at least one key is required")
	}

	for i, key := range c.Paths {
		if err := key.Validate(); err != nil {
			return fmt.Errorf("key %d is invalid: %w", i, err)
		}
	}

	return nil
}

type signingFileKeyConfig struct {
	Path     string            `koanf:"path" help:"The path to the key"`
	Generate signing.Algorithm `koanf:"generate" help:"The algorithm used to generate the key, if it doesn't exist"`
}

func (c *signingFileKeyConfig) Validate() error {
	if len(c.Path) == 0 {
		return fmt.Errorf("a path is required")
	}

	if len(c.Generate) > 0 && !signing.ValidAlgorithm(c.Generate) {
		return fmt.Errorf("invalid signing algorithm %s", c.Generate)
	}

	return nil
}

var (
	fileMapPathKey     = reflect.ValueOf("path")
	fileMapGenerateKey = reflect.ValueOf("generate")
)

func (c *signingFileKeyConfig) UnmarshalMapstructure(input any) error {
	in := reflect.ValueOf(input)
	switch in.Kind() {
	case reflect.String:
		c.Path = in.String()
		return nil

	case reflect.Map:
		pathValue := in.MapIndex(fileMapPathKey)
		if path, ok := reflect.TypeAssert[string](pathValue); ok {
			c.Path = path
		} else {
			inner := reflect.ValueOf(pathValue.Interface())
			return fmt.Errorf("expected string for key 'path', got %s", inner.Kind())
		}

		generateValue := in.MapIndex(fileMapGenerateKey)
		if generate, ok := reflect.TypeAssert[string](generateValue); ok {
			c.Generate = signing.Algorithm(generate)
		} else {
			inner := reflect.ValueOf(generateValue.Interface())
			return fmt.Errorf("expected string for key 'generate', got %s", inner.Kind())
		}

		return nil

	default:
		return fmt.Errorf("expected map or string, got %s", in.Kind())
	}
}

type signingMemoryConfig struct {
	RS256 bool `koanf:"rs256" help:"Generate a RSA 2048-bit key"`
	RS384 bool `koanf:"rs384" help:"Generate a RSA 3072-bit key"`
	RS512 bool `koanf:"rs512" help:"Generate a RSA 4096-bit key"`
	ES256 bool `koanf:"es256" help:"Generate an ECDSA curve P-256 key"`
	ES384 bool `koanf:"es384" help:"Generate an ECDSA curve P-384 key"`
	ES512 bool `koanf:"es512" help:"Generate an ECDSA curve P-521 key"`
}

func (c *signingMemoryConfig) Validate() error {
	if c.RS256 || c.RS384 || c.RS512 || c.ES256 || c.ES384 || c.ES512 {
		return nil
	} else {
		return errors.New("at least one signing algorithm type must be enabled")
	}
}

func (c *signingMemoryConfig) Algorithms() map[signing.Algorithm]bool {
	return map[signing.Algorithm]bool{
		signing.RS256: c.RS256,
		signing.RS384: c.RS384,
		signing.RS512: c.RS512,
		signing.ES256: c.ES256,
		signing.ES384: c.ES384,
		signing.ES512: c.ES512,
	}
}

type tailscaleConfig struct {
	Hostname        string                         `koanf:"hostname" help:"The hostname to advertise as"`
	StateDirectory  string                         `koanf:"state-directory" help:"Where to store Tailscale connection state"`
	Tags            []string                       `koanf:"tags" help:"The tags to register for the node"`
	Authentication  tailscale.AuthenticationMethod `koanf:"authentication"`
	ControlPlaneUrl string                         `koanf:"control-plane-url" help:"The Tailscale/Headscale control plane to connect to"`

	AuthKey    string                    `koanf:"authkey" help:"The auth key used to join the tailnet"`
	OAuth      tailscaleOauthConfig      `koanf:"oauth"`
	Federation tailscaleFederationConfig `koanf:"federation"`
}

func (c *tailscaleConfig) SetDefaults() {
	c.Hostname = "tailfed-server"
	c.StateDirectory = ""
	c.Authentication = tailscale.AuthenticationMethodKey
	c.ControlPlaneUrl = ipn.DefaultControlURL
}

func (c *tailscaleConfig) Validate() error {
	switch c.Authentication {
	case tailscale.AuthenticationMethodKey:
		if len(c.AuthKey) == 0 {
			return errors.New("an authkey is required")
		} else {
			return nil
		}

	case tailscale.AuthenticationMethodOAuth:
		return c.OAuth.Validate()

	case tailscale.AuthenticationMethodFederation:
		return c.Federation.Validate()

	default:
		return fmt.Errorf("unknown authentication method %q (options: %s)", c.Authentication, tailscale.AuthenticationMethods)
	}
}

func (c *tailscaleConfig) Connect(ctx context.Context) (tailscale.Connection, error) {
	return tailscale.NewInProcess(ctx, c.Hostname, c.StateDirectory, c.ControlPlaneUrl, c.Tags, c.credentials())
}

func (c *tailscaleConfig) credentials() tailscale.CredentialProvider {
	switch c.Authentication {
	case tailscale.AuthenticationMethodKey:
		return tailscale.AuthKey(c.AuthKey)
	case tailscale.AuthenticationMethodOAuth:
		return tailscale.OAuth(c.OAuth.ClientId, c.OAuth.ClientSecret)
	case tailscale.AuthenticationMethodFederation:
		return tailscale.WorkloadIdentityFederation(c.Federation.ClientId, c.Federation.IdToken, c.Federation.Audience)
	default:
		panic("unreachable")
	}
}

type tailscaleOauthConfig struct {
	ClientId     string `koanf:"client-id" help:"The OAuth app client ID"`
	ClientSecret string `koanf:"client-secret" help:"The OAuth app client secret"`
}

func (c *tailscaleOauthConfig) Validate() error {
	switch {
	case len(c.ClientId) == 0:
		return errors.New("a client ID is required")
	case len(c.ClientSecret) == 0:
		return errors.New("a client secret is required")
	default:
		return nil
	}
}

type tailscaleFederationConfig struct {
	ClientId string `koanf:"client-id" help:"The OpenID connect client ID"`
	IdToken  string `koanf:"id-token" help:"The identity provider's existing ID token"`
	Audience string `koanf:"audience" help:"The audience used to automatically fetch an ID token"`
}

func (c *tailscaleFederationConfig) Validate() error {
	switch {
	case len(c.ClientId) == 0:
		return errors.New("a client ID is required")
	case len(c.IdToken) == 0 && len(c.Audience) == 0:
		return errors.New("an ID token or audience is required")
	default:
		return nil
	}
}

var profileIdRegex = regexp.MustCompile("^[a-z0-9]+$")

type tokenConfig struct {
	DefaultProfile string                  `koanf:"default-profile" help:"The default profile to use when none is requested"`
	Profiles       map[string]oidc.Profile `koanf:"profiles"`
}

func (c *tokenConfig) SetDefaults() {
	var p oidc.Profile
	p.SetDefaults()

	c.DefaultProfile = "default"
	c.Profiles = map[string]oidc.Profile{c.DefaultProfile: p}
}

func (c *tokenConfig) Validate() error {
	for id, profile := range c.Profiles {
		if !profileIdRegex.MatchString(id) {
			return fmt.Errorf("profile id %q must contain only lowercase alphanumeric characters", id)
		}

		if err := profile.Validate(); err != nil {
			return fmt.Errorf("invalid profile %q: %w", id, err)
		}
	}

	if _, ok := c.Profiles[c.DefaultProfile]; !ok {
		return fmt.Errorf("default profile %q must exist in profiles", c.DefaultProfile)
	}

	return nil
}
