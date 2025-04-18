package signing

import (
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// Backend provides a mechanism for generating signed JWTs
type Backend interface {
	// PublicKey returns details about the public key
	PublicKey() (jose.JSONWebKey, error)
	// Sign generates a signed JWT with the provided claims
	Sign(claims Claims) (string, error)
}

// Claims contains the data that will be signed in the token
type Claims struct {
	jwt.Claims
}

// NewClaims creates a new set of token claims for signing
func NewClaims(issuer, audience, host string, validity time.Duration) Claims {
	now := time.Now()

	return Claims{
		Claims: jwt.Claims{
			Issuer:    issuer,
			Audience:  jwt.Audience{audience},
			Subject:   host,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Expiry:    jwt.NewNumericDate(now.Add(validity)),
		},
	}
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
