package signing

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-jose/go-jose/v4"
	"github.com/sirupsen/logrus"
)

// NewInMemory creates a new signer with in-memory ephemeral keys
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

		generic, err := toLocalKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to create signer for %q: %w", algorithm, err)
		}

		logger.WithField("key.algorithm", algorithm).WithField("key.id", generic.publicKey.KeyID).Debug("generated new private key")
		keys = append(keys, generic)
	}

	logger.WithField("memory.keys", len(keys)).Info("created new in-memory signer(s)")
	return generic(keys), nil
}

// File is a private key used for signing
type File struct {
	Path     string
	Generate Algorithm
}

// NewFile create a new signer with filesystem-backed keys
func NewFile(files []File) (Backend, error) {
	logger := logrus.WithField("component", "signer").WithField("signer.backend", "file")

	keys := []genericKey{}
	for _, file := range files {
		var key crypto.Signer

		fileLogger := logger.WithField("key.path", file.Path)

		fileLogger.Debug("loading private key")
		bytes, err := os.ReadFile(file.Path)
		if err == nil {
			key, err = parsePrivateKey(bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s: %w", file.Path, err)
			}
			fileLogger.Info("loaded private key")
		} else if len(file.Generate) > 0 && os.IsNotExist(err) {
			fileLogger.
				WithField("key.algorithm", file.Generate).
				Info("generating a new key")

			key, err = generateKey(file.Generate)
			if err != nil {
				return nil, fmt.Errorf("failed to generate %s key for %s: %w", file.Generate, file.Path, err)
			}

			bytes, err = x509.MarshalPKCS8PrivateKey(key)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal new key for %s: %w", file.Path, err)
			}

			if err := os.WriteFile(file.Path, bytes, 0o600); err != nil {
				return nil, fmt.Errorf("failed to write new key for %s: %w", file.Path, err)
			}
		} else {
			return nil, fmt.Errorf("failed to read key %s: %w", file.Path, err)
		}

		if generic, err := toLocalKey(key); err == nil {
			keys = append(keys, generic)
			fileLogger.WithField("key.algorithm", generic.algorithm).Info("validated key")
		} else {
			return nil, fmt.Errorf("failed to create signer for %s: %w", file.Path, err)
		}
	}

	logger.WithField("file.keys", len(keys)).Info("created new file signer")
	return generic(keys), nil
}

func generateKey(algorithm Algorithm) (crypto.Signer, error) {
	switch algorithm {
	case RS256:
		return rsa.GenerateKey(rand.Reader, 2048)
	case RS384:
		return rsa.GenerateKey(rand.Reader, 3072)
	case RS512:
		return rsa.GenerateKey(rand.Reader, 4096)
	case ES256:
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case ES384:
		return ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case ES512:
		return ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	default:
		panic("unreachable")
	}
}

func toLocalKey(k crypto.Signer) (genericKey, error) {
	id, err := fingerprint(k.Public())
	if err != nil {
		return genericKey{}, err
	}

	switch private := k.(type) {
	case *rsa.PrivateKey:
		signer, err := newKey(id, private, RS256)
		if err != nil {
			return genericKey{}, err
		}

		return genericKey{
			algorithm: RS256,
			signer:    signer,
			publicKey: jose.JSONWebKey{
				Use:       "sig",
				KeyID:     id,
				Key:       private.Public(),
				Algorithm: string(toJose[RS256]),
			},
		}, nil

	case *ecdsa.PrivateKey:
		var algorithm Algorithm
		switch private.Curve {
		case elliptic.P256():
			algorithm = ES256
		case elliptic.P384():
			algorithm = ES384
		case elliptic.P521():
			algorithm = ES512
		}

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
				Key:       private.Public(),
				Algorithm: string(toJose[algorithm]),
			},
		}, nil

	default:
		panic("unreachable")
	}
}

func fingerprint(pub crypto.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:]), nil
}

func parsePrivateKey(data []byte) (crypto.Signer, error) {
	if !bytes.Contains(data, []byte("-----BEGIN")) {
		return parseDER(data)
	}

	rest := data
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return nil, errors.New("no private key PEM block found")
		}

		if block.Type == "ENCRYPTED PRIVATE KEY" || block.Headers["Proc-Type"] == "4.ENCRYPTED" {
			return nil, errors.New("encrypted private keys are not supported")
		}

		if strings.HasSuffix(block.Type, "PRIVATE KEY") {
			return parseDER(block.Bytes)
		}
	}
}

func parseDER(der []byte) (crypto.Signer, error) {
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch k := key.(type) {
		case *rsa.PrivateKey:
			return k, nil
		case *ecdsa.PrivateKey:
			return k, nil
		default:
			return nil, fmt.Errorf("unsupported private key type %T", k)
		}
	}

	if k, err := x509.ParseECPrivateKey(der); err == nil {
		return k, nil
	}
	if k, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return k, nil
	}

	return nil, errors.New("unrecognized or invalid private key format")
}
