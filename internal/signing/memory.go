package signing

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// inMemory generates ephemeral private and public keys for signing
type inMemory struct {
	id      string
	private *rsa.PrivateKey
}

var _ Backend = (*inMemory)(nil)

// NewInMemory creates a new keys with in-memory ephemeral keys
func NewInMemory() (Backend, error) {
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	der, err := x509.MarshalPKIXPublicKey(&private.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encode public key to der: %w", err)
	}
	fingerprint := sha1.Sum(der)
	id := hex.EncodeToString(fingerprint[:])

	return &inMemory{id, private}, nil
}

func (m *inMemory) Sign(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = m.id
	return token.SignedString(m.private)
}

func (m *inMemory) PublicKeys() (map[string]crypto.PublicKey, error) {
	return map[string]crypto.PublicKey{m.id: m.private.PublicKey}, nil
}
