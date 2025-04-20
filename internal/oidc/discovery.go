package oidc

import "github.com/go-jose/go-jose/v4"

var (
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
		Claims:            claimsKeys(),
		ResponseTypes:     responseTypes,
		SigningAlgorithms: signingAlgorithms,
		SubjectTypes:      subjectTypes,
	}
}
