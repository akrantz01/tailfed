package signing

import (
	"github.com/akrantz01/tailfed/internal/oidc"
	"github.com/go-jose/go-jose/v4"
)

// Backend provides a mechanism for generating signed JWTs
type Backend interface {
	// PublicKey returns details about the public key
	PublicKey() (jose.JSONWebKey, error)
	// Sign generates a signed JWT with the provided claims
	Sign(claims oidc.Claims) (string, error)
}

// newKey creates a new [jose.Signer] for generating new JWTs with an embedded key ID in the header
func newKey(keyId string, key any, algorithm jose.SignatureAlgorithm) (jose.Signer, error) {
	return jose.NewSigner(
		jose.SigningKey{Key: key, Algorithm: algorithm},
		&jose.SignerOptions{
			ExtraHeaders: map[jose.HeaderKey]any{"kid": keyId},
		},
	)
}
