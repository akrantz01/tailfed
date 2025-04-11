package signing

import (
	"crypto"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Backend provides a mechanism for generating signed JWTs
type Backend interface {
	// PublicKeys returns a mapping of key ID to public key for JWK set generation
	PublicKeys() map[string]crypto.PublicKey
	// Sign generates a signed JWT with the provided claims
	Sign(claims jwt.Claims) (string, error)
}

type claims struct {
	jwt.RegisteredClaims
}

var _ jwt.Claims = (*claims)(nil)

// NewClaims creates a new set of token claims for signing
func NewClaims(domain, audience, host string, validity time.Duration) jwt.Claims {
	now := time.Now()

	return &claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://" + domain,
			Audience:  jwt.ClaimStrings{audience},
			Subject:   host,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(validity)),
		},
	}
}
