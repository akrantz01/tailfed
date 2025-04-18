package metadata

import (
	"context"

	"github.com/go-jose/go-jose/v4"
)

// Backend provides a mechanism for storing OpenID Connect metadata
type Backend interface {
	// Load reads and deserializes metadata from JSON. Only used for local development
	Load(ctx context.Context, key string, out any) error
	// Save stores a new set of metadata. The metadata must be serializable to JSON
	Save(ctx context.Context, key string, data any) error
}

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
		Issuer:  issuer,
		JwksUri: issuer + "/.well-known/jwks.json",
		Claims: []string{
			"aud", "iat", "iss", "sub", "exp", "nbf",
			"tailnet", "dns_name", "machine_name", "host_name",
			"os", "tags", "authorized", "external",
		},
		ResponseTypes: []string{"id_token"},
		SigningAlgorithms: []jose.SignatureAlgorithm{
			jose.RS256, jose.RS384, jose.RS512,
			jose.ES256, jose.ES384, jose.ES256,
		},
		SubjectTypes: []string{"public"},
	}
}
