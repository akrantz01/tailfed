package signing

import (
	"strings"

	"github.com/go-jose/go-jose/v4"
)

// BackendId identifies a backend in the configuration
type BackendId string

const (
	BackendMemory BackendId = "memory"
	BackendAWSKMS BackendId = "aws-kms"
)

func (b BackendId) Choices() []string {
	return []string{string(BackendMemory), string(BackendAWSKMS)}
}

// NeedsAWS returns true when the backend needs an AWS config
func (b BackendId) NeedsAWS() bool {
	return b == BackendAWSKMS
}

var BackendIds string = strings.Join(BackendMemory.Choices(), ", ")

// Algorithm is an algorithm that can be used for signing tokens
type Algorithm string

const (
	RS256 = Algorithm("RS256")
	RS384 = Algorithm("RS384")
	RS512 = Algorithm("RS512")
	ES256 = Algorithm("ES256")
	ES384 = Algorithm("ES384")
	ES512 = Algorithm("ES512")
)

var toJose = map[Algorithm]jose.SignatureAlgorithm{
	RS256: jose.RS256,
	RS384: jose.RS384,
	RS512: jose.RS512,
	ES256: jose.ES256,
	ES384: jose.ES384,
	ES512: jose.ES512,
}

func (a Algorithm) Choices() []string {
	return []string{
		string(RS256),
		string(RS384),
		string(RS512),
		string(ES256),
		string(ES384),
		string(ES512),
	}
}

// Backend provides a mechanism for generating signed JWTs
type Backend interface {
	// PublicKeys returns details about the available public keys
	PublicKeys() []jose.JSONWebKey
	// Sign generates a signed JWT with the provided claims
	Sign(algorithm Algorithm, claims any) (string, error)
}

// newKey creates a new [jose.Signer] for generating new JWTs with an embedded key ID in the header
func newKey(keyId string, key any, algorithm Algorithm) (jose.Signer, error) {
	return jose.NewSigner(
		jose.SigningKey{Key: key, Algorithm: toJose[algorithm]},
		&jose.SignerOptions{
			ExtraHeaders: map[jose.HeaderKey]any{"kid": keyId},
		},
	)
}
