package signing

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"

	"github.com/akrantz01/tailfed/internal/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// inMemory generates ephemeral private and public keys for signing
type inMemory struct {
	id      string
	private *rsa.PrivateKey
	signer  jose.Signer
}

var _ Backend = (*inMemory)(nil)

// NewInMemory creates a new keys with in-memory ephemeral keys
func NewInMemory(logger logrus.FieldLogger) (Backend, error) {
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	id := uuid.Must(uuid.NewV7()).String()
	logger.WithField("id", id).Debug("generated new RS256 private key")

	signer, err := newKey(id, private, jose.RS256)
	if err != nil {
		return nil, err
	}

	logger.Info("created new in-memory signer")
	return &inMemory{id, private, signer}, nil
}

func (m *inMemory) Sign(claims oidc.Claims) (string, error) {
	return jwt.Signed(m.signer).Claims(claims).Serialize()
}

func (m *inMemory) PublicKey() (jose.JSONWebKey, error) {
	return jose.JSONWebKey{
		Use:       "sig",
		KeyID:     m.id,
		Key:       &m.private.PublicKey,
		Algorithm: string(jose.RS256),
	}, nil
}
