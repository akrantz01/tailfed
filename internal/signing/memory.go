package signing

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"fmt"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// inMemory generates ephemeral private and public keys for signing
type inMemory struct {
	keys []memoryKey
}

var _ Backend = (*inMemory)(nil)

// NewInMemory creates a new keys with in-memory ephemeral keys
func NewInMemory(algorithms map[Algorithm]bool) (Backend, error) {
	logger := logrus.WithField("component", "signer").WithField("signer.backend", "memory")

	keys := []memoryKey{}
	for algorithm, enabled := range algorithms {
		if !enabled {
			continue
		}

		key, err := generateKey(algorithm)
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key for %q: %w", algorithm, err)
		}

		logger.WithField("key.algorithm", algorithm).WithField("key.id", key.public.KeyID).Debug("generated new private key")
		keys = append(keys, key)
	}

	logger.WithField("memory.keys", len(keys)).Info("created new in-memory signer(s)")
	return &inMemory{keys}, nil
}

func (m *inMemory) Sign(algorithm Algorithm, claims any) (string, error) {
	for _, key := range m.keys {
		if key.algorithm == algorithm {
			return jwt.Signed(key.signer).Claims(claims).Serialize()
		}
	}
	return "", fmt.Errorf("unsupported signing algorithm %q", algorithm)
}

func (m *inMemory) PublicKeys() []jose.JSONWebKey {
	publicKeys := make([]jose.JSONWebKey, 0, len(m.keys))
	for _, key := range m.keys {
		publicKeys = append(publicKeys, key.public)
	}
	return publicKeys
}

type memoryKey struct {
	algorithm Algorithm
	public    jose.JSONWebKey
	signer    jose.Signer
}

func generateKey(algorithm Algorithm) (memoryKey, error) {
	switch algorithm {
	case RS256:
		private, err := rsa.GenerateKey(rand.Reader, 2048)
		return toMemoryKey(algorithm, private, err, rsaPublicKey)
	case RS384:
		private, err := rsa.GenerateKey(rand.Reader, 3072)
		return toMemoryKey(algorithm, private, err, rsaPublicKey)
	case RS512:
		private, err := rsa.GenerateKey(rand.Reader, 4096)
		return toMemoryKey(algorithm, private, err, rsaPublicKey)
	case ES256:
		private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		return toMemoryKey(algorithm, private, err, ecdsaPublicKey)
	case ES384:
		private, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		return toMemoryKey(algorithm, private, err, ecdsaPublicKey)
	case ES512:
		private, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		return toMemoryKey(algorithm, private, err, ecdsaPublicKey)
	default:
		return memoryKey{}, fmt.Errorf("unsupported algorithm %q", algorithm)
	}
}

func toMemoryKey[Priv, Pub any](algorithm Algorithm, private Priv, err error, getPublicKey func(k Priv) Pub) (memoryKey, error) {
	if err != nil {
		return memoryKey{}, err
	}

	id := uuid.Must(uuid.NewV7()).String()
	signer, err := newKey(id, private, algorithm)
	if err != nil {
		return memoryKey{}, err
	}

	return memoryKey{
		algorithm: algorithm,
		signer:    signer,
		public: jose.JSONWebKey{
			Use:       "sig",
			KeyID:     id,
			Key:       getPublicKey(private),
			Algorithm: string(toJose[algorithm]),
		},
	}, nil
}

func rsaPublicKey(k *rsa.PrivateKey) *rsa.PublicKey {
	return &k.PublicKey
}

func ecdsaPublicKey(k *ecdsa.PrivateKey) *ecdsa.PublicKey {
	return &k.PublicKey
}
