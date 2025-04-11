package signing

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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

	id := uuid.Must(uuid.NewV7()).String()
	return &inMemory{id, private}, nil
}

func (m *inMemory) Sign(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = m.id
	return token.SignedString(m.private)
}

func (m *inMemory) PublicKeys() map[string]crypto.PublicKey {
	return map[string]crypto.PublicKey{m.id: m.private.PublicKey}
}
