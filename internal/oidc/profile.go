package oidc

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/akrantz01/tailfed/internal/signing"
)

// Profiles contains all the registered profiles
type Profiles struct {
	defaultId string
	profiles  map[string]Profile
}

// NewProfiles creates a new profile directory
func NewProfiles(defaultId string, profiles map[string]Profile) Profiles {
	return Profiles{
		defaultId: defaultId,
		profiles:  profiles,
	}
}

// Get a profile by ID
func (p *Profiles) Get(id string) *Profile {
	if profile, ok := p.profiles[id]; ok {
		return &profile
	} else {
		return nil
	}
}

// GetDefault returns the default profile
func (p *Profiles) GetDefault() *Profile {
	d := p.profiles[p.defaultId]
	return &d
}

// Profile is a client-selectable configuration for a token
type Profile struct {
	Audience           string              `koanf:"audience"`
	AcceptedAlgorithms []signing.Algorithm `koanf:"accepted-algorithms"`
	MaxSubjectLength   uint                `koanf:"max-subject-length"`
	Subject            SubjectConfig       `koanf:"subject"`
	Validity           time.Duration       `koanf:"validity"`
}

func (p *Profile) SetDefaults() {
	p.Audience = "tailfed"
	p.AcceptedAlgorithms = []signing.Algorithm{
		signing.RS256,
		signing.RS384,
		signing.RS512,
		signing.ES256,
		signing.ES384,
		signing.ES512,
	}
	p.Validity = 1 * time.Hour
}

func (p *Profile) Validate() error {
	if len(strings.TrimSpace(p.Audience)) == 0 {
		return errors.New("a non-empty audience is required")
	}
	if len(p.AcceptedAlgorithms) == 0 {
		return errors.New("at least one signing algorithm must be accepted")
	}

	if err := p.Subject.Validate(); err != nil {
		return fmt.Errorf("invalid subject configuration: %w", err)
	}

	if p.Validity <= 0 {
		return errors.New("token validity must be positive")
	}

	return nil
}

// SubjectConfig defines what should be included in the `sub` claim depending on device owner
type SubjectConfig struct {
	Tag  []string `koanf:"tag"`
	User []string `koanf:"user"`
}

func (s *SubjectConfig) Validate() error {
	if len(s.Tag) == 0 {
		return errors.New("a format for tag-owned devices is required")
	}
	if len(s.User) == 0 {
		return errors.New("a format for user-owned devices is required")
	}

	return nil
}
