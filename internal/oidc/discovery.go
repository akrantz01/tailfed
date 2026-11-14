package oidc

import (
	"reflect"
	"strings"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

var (
	claims            = structKeys(reflect.TypeFor[jwt.Claims]())
	responseTypes     = []string{"id_token"}
	subjectTypes      = []string{"public"}
	signingAlgorithms = []jose.SignatureAlgorithm{
		jose.RS256, jose.RS384, jose.RS512,
		jose.ES256, jose.ES384, jose.ES256,
	}
)

// DiscoveryDocument is an OpenID Connect discovery document for an AWS IAM identity to read
type DiscoveryDocument struct {
	Issuer            string                    `json:"issuer"`
	JwksUri           string                    `json:"jwks_uri"`
	Claims            []string                  `json:"claims_supported"`
	ResponseTypes     []string                  `json:"response_types_supported"`
	SigningAlgorithms []jose.SignatureAlgorithm `json:"id_token_signing_alg_values_supported"`
	SubjectTypes      []string                  `json:"subject_types_supported"`
}

// NewDiscoveryDocument creates a new OpenID Connect discovery document from an issuer URL
func NewDiscoveryDocument(issuer string) DiscoveryDocument {
	return DiscoveryDocument{
		Issuer:            issuer,
		JwksUri:           issuer + "/.well-known/jwks.json",
		Claims:            claims,
		ResponseTypes:     responseTypes,
		SigningAlgorithms: signingAlgorithms,
		SubjectTypes:      subjectTypes,
	}
}

func structKeys(t reflect.Type) []string {
	var keys []string
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Anonymous {
			keys = append(keys, structKeys(f.Type)...)
		} else if value, ok := f.Tag.Lookup("json"); ok {
			parts := strings.Split(value, ",")
			keys = append(keys, parts[0])
		}
	}

	return keys
}
