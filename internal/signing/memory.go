package signing

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"fmt"

	"github.com/go-jose/go-jose/v4"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// NewInMemory creates a new keys with in-memory ephemeral keys
func NewInMemory(algorithms map[Algorithm]bool) (Backend, error) {
	logger := logrus.WithField("component", "signer").WithField("signer.backend", "memory")

	keys := []genericKey{}
	for algorithm, enabled := range algorithms {
		if !enabled {
			continue
		}

		key, err := generateKey(algorithm)
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key for %q: %w", algorithm, err)
		}

		logger.WithField("key.algorithm", algorithm).WithField("key.id", key.publicKey.KeyID).Debug("generated new private key")
		keys = append(keys, key)
	}

	logger.WithField("memory.keys", len(keys)).Info("created new in-memory signer(s)")
	return generic(keys), nil
}

func generateKey(algorithm Algorithm) (genericKey, error) {
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
		return genericKey{}, fmt.Errorf("unsupported algorithm %q", algorithm)
	}
}

func toMemoryKey[Priv, Pub any](algorithm Algorithm, private Priv, err error, getPublicKey func(k Priv) Pub) (genericKey, error) {
	if err != nil {
		return genericKey{}, err
	}

	id := uuid.Must(uuid.NewV7()).String()
	signer, err := newKey(id, private, algorithm)
	if err != nil {
		return genericKey{}, err
	}

	return genericKey{
		algorithm: algorithm,
		signer:    signer,
		publicKey: jose.JSONWebKey{
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
