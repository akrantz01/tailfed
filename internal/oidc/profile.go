package oidc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/go-jose/go-jose/v4/jwt"
)

var ErrSubjectTooLong = errors.New("subject exceeds max length")

// SubjectField is a part of the node that can be included in the subject
type SubjectField string

const (
	SubjectFieldHostname    = SubjectField("hostname")
	SubjectFieldId          = SubjectField("id")
	SubjectFieldTailnetId   = SubjectField("tailnet.id")
	SubjectFieldTailnetName = SubjectField("tailnet.name")
	SubjectFieldTailnetDns  = SubjectField("tailnet.dns")
	SubjectFieldTags        = SubjectField("tags")
	SubjectFieldUserId      = SubjectField("user.id")
	SubjectFieldUserLogin   = SubjectField("user.login")
	SubjectFieldUserGroups  = SubjectField("user.groups")
)

var (
	subjectFields = strings.Join([]string{
		string(SubjectFieldHostname),
		string(SubjectFieldId),
		string(SubjectFieldTailnetId),
		string(SubjectFieldTailnetName),
		string(SubjectFieldTailnetDns),
		string(SubjectFieldTags),
		string(SubjectFieldUserId),
		string(SubjectFieldUserLogin),
		string(SubjectFieldUserGroups),
	}, ", ")
	knownSubjectFields = map[SubjectField]bool{
		SubjectFieldHostname:    true,
		SubjectFieldId:          true,
		SubjectFieldTailnetId:   true,
		SubjectFieldTailnetName: true,
		SubjectFieldTailnetDns:  true,
		SubjectFieldTags:        true,
		SubjectFieldUserId:      true,
		SubjectFieldUserLogin:   true,
		SubjectFieldUserGroups:  true,
	}
)

func (f SubjectField) availableFor(tagOwned bool) bool {
	switch f {
	case SubjectFieldTags:
		return tagOwned
	case SubjectFieldUserId, SubjectFieldUserLogin, SubjectFieldUserGroups:
		return !tagOwned
	default:
		return true
	}
}

func (f SubjectField) resolveFor(node *tailscale.Node) string {
	switch f {
	case SubjectFieldHostname:
		return node.Hostname
	case SubjectFieldId:
		return node.ID
	case SubjectFieldTailnetId:
		return node.Tailnet.ID
	case SubjectFieldTailnetName:
		return node.Tailnet.Name
	case SubjectFieldTailnetDns:
		return node.Tailnet.DNS
	case SubjectFieldTags:
		return strings.Join(node.Tags, ",")
	case SubjectFieldUserId:
		return strconv.FormatInt(node.User.ID, 10)
	case SubjectFieldUserLogin:
		return node.User.Login
	case SubjectFieldUserGroups:
		return strings.Join(node.User.Groups, ",")
	default:
		panic("unreachable")
	}
}

// Profiles contains all the registered profiles
type Profiles struct {
	defaultId      string
	defaultProfile Profile
	allProfiles    map[string]Profile
}

// NewProfiles creates a new profile directory
func NewProfiles(defaultId string, profiles map[string]Profile) Profiles {
	return Profiles{
		defaultId:      defaultId,
		defaultProfile: profiles[defaultId],
		allProfiles:    profiles,
	}
}

// GetOrDefault gets a profile by ID. If no ID is provided, the default is returned
func (p *Profiles) GetOrDefault(id string) (string, *Profile) {
	if len(id) == 0 {
		return p.defaultId, &p.defaultProfile
	} else if profile, ok := p.allProfiles[id]; ok {
		return id, &profile
	} else {
		return "", nil
	}
}

// Profile is a client-selectable configuration for a token
type Profile struct {
	Audience           string              `koanf:"audience"`
	AcceptedAlgorithms []signing.Algorithm `koanf:"accepted-algorithms"`
	MaxSubjectLength   uint                `koanf:"max-subject-length"`
	Subject            SubjectConfig       `koanf:"subject"`
	Validity           time.Duration       `koanf:"validity"`
}

// GenerateClaims creates a new JWT claims set for the device
func (p *Profile) GenerateClaims(issuer string, node *tailscale.Node) (jwt.Claims, error) {
	subject, err := p.Subject.Render(node, p.MaxSubjectLength)
	if err != nil {
		return jwt.Claims{}, err
	}

	now := time.Now()
	return jwt.Claims{
		Issuer:   issuer,
		Subject:  subject,
		Audience: []string{p.Audience},
		IssuedAt: jwt.NewNumericDate(now),
		Expiry:   jwt.NewNumericDate(now.Add(p.Validity)),
	}, nil
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
	Tag  []SubjectField `koanf:"tag"`
	User []SubjectField `koanf:"user"`
}

// Render a subject for a particular node
func (s *SubjectConfig) Render(node *tailscale.Node, maxLength uint) (string, error) {
	fields := s.fieldsFor(node)
	var subject strings.Builder

	for i, field := range fields {
		if i > 0 {
			_, _ = subject.WriteRune(':')
		}
		_, _ = subject.WriteString(string(field))
		_, _ = subject.WriteRune(':')
		_, _ = subject.WriteString(field.resolveFor(node))
	}

	rendered := subject.String()
	if maxLength != 0 && uint(len(rendered)) > maxLength {
		return "", ErrSubjectTooLong
	} else {
		return rendered, nil
	}
}

func (s *SubjectConfig) fieldsFor(node *tailscale.Node) []SubjectField {
	if node.TagOwned() {
		return s.Tag
	} else {
		return s.User
	}
}

func (s *SubjectConfig) SetDefaults() {
	s.Tag = []SubjectField{SubjectFieldTailnetDns, SubjectFieldHostname, SubjectFieldTags}
	s.User = []SubjectField{SubjectFieldTailnetDns, SubjectFieldHostname, SubjectFieldUserLogin}
}

func (s *SubjectConfig) Validate() error {
	if len(s.Tag) == 0 {
		return errors.New("a format for tag-owned devices is required")
	}
	if err := validateFields(s.Tag, true); err != nil {
		return fmt.Errorf("invalid tag-owned fields: %w", err)
	}

	if len(s.User) == 0 {
		return errors.New("a format for user-owned devices is required")
	}
	if err := validateFields(s.User, false); err != nil {
		return fmt.Errorf("invalid user-own fields: %w", err)
	}

	return nil
}

func validateFields(fields []SubjectField, tagOwned bool) error {
	seen := make(map[SubjectField]bool, len(fields))
	for _, field := range fields {
		if _, ok := knownSubjectFields[field]; !ok {
			return fmt.Errorf("unknown field %q (options %s)", field, subjectFields)
		}

		if seen[field] {
			return fmt.Errorf("duplicate field %q", field)
		}
		seen[field] = true

		if !field.availableFor(tagOwned) {
			return fmt.Errorf("field %q is unavailable", field)
		}
	}

	return nil
}
