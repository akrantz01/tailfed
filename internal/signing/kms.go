package signing

import (
	"context"
	"crypto"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/golang-jwt/jwt/v5"
)

// kmsBackend signs tokens using a key stored in AWS KMS
type kmsBackend struct {
	id            string
	arn           string
	client        *kms.Client
	signingMethod jwt.SigningMethod
}

var _ Backend = (*kmsBackend)(nil)

// NewKMS creates a new AWS KMS-backed signer
func NewKMS(config aws.Config, alias string) (Backend, error) {
	client := kms.NewFromConfig(config)

	details, err := client.DescribeKey(context.Background(), &kms.DescribeKeyInput{KeyId: aws.String(alias)})
	if err != nil {
		return nil, fmt.Errorf("failed to describe key: %w", err)
	}
	metadata := details.KeyMetadata

	if !metadata.Enabled {
		return nil, errors.New("key is not enabled")
	}
	if metadata.KeyUsage != types.KeyUsageTypeSignVerify {
		return nil, errors.New("key is not configured for sign/verify")
	}

	signingMethod, err := kmsSigningMethodFromSpec(metadata.KeySpec, client)
	if err != nil {
		return nil, err
	}

	return &kmsBackend{*metadata.KeyId, *metadata.Arn, client, signingMethod}, nil
}

func (k *kmsBackend) Sign(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(k.signingMethod, claims)
	token.Header["kid"] = k.id
	return token.SignedString(k.arn)
}

func (k *kmsBackend) PublicKeys() map[string]crypto.PublicKey {
	return nil
}

type kmsSigningMethod struct {
	Name        string
	Hash        types.SigningAlgorithmSpec
	Transformer func([]byte) ([]byte, error)
	client      *kms.Client
}

var _ jwt.SigningMethod = (*kmsSigningMethod)(nil)

func kmsSigningMethodFromSpec(spec types.KeySpec, client *kms.Client) (jwt.SigningMethod, error) {
	switch spec {
	case types.KeySpecRsa2048:
		return &kmsSigningMethod{"RS256", types.SigningAlgorithmSpecRsassaPkcs1V15Sha256, nil, client}, nil
	case types.KeySpecRsa3072:
		return &kmsSigningMethod{"RS384", types.SigningAlgorithmSpecRsassaPkcs1V15Sha384, nil, client}, nil
	case types.KeySpecRsa4096:
		return &kmsSigningMethod{"RS512", types.SigningAlgorithmSpecRsassaPkcs1V15Sha512, nil, client}, nil
	case types.KeySpecEccNistP256:
		return &kmsSigningMethod{"ES256", types.SigningAlgorithmSpecEcdsaSha256, transformEcdsaSignature(32), client}, nil
	case types.KeySpecEccNistP384:
		return &kmsSigningMethod{"ES384", types.SigningAlgorithmSpecEcdsaSha384, transformEcdsaSignature(48), client}, nil
	case types.KeySpecEccNistP521:
		return &kmsSigningMethod{"ES512", types.SigningAlgorithmSpecEcdsaSha512, transformEcdsaSignature(66), client}, nil
	default:
		return nil, fmt.Errorf("unsupported key type %q", spec)
	}
}

func (m *kmsSigningMethod) Alg() string {
	return m.Name
}

func (m *kmsSigningMethod) Sign(signingString string, key interface{}) ([]byte, error) {
	id, ok := key.(string)
	if !ok {
		return nil, fmt.Errorf("key is of invalid type: KMS sign expects string")
	}

	output, err := m.client.Sign(context.Background(), &kms.SignInput{
		KeyId:            aws.String(id),
		Message:          []byte(signingString),
		MessageType:      types.MessageTypeRaw,
		SigningAlgorithm: m.Hash,
	})
	if err != nil {
		return nil, err
	}

	signature := output.Signature
	if m.Transformer != nil {
		var err error
		if signature, err = m.Transformer(signature); err != nil {
			return nil, fmt.Errorf("failed to transform signature to JWT-compatible format: %w", err)
		}
	}

	return signature, nil
}

func (m *kmsSigningMethod) Verify(string, []byte, interface{}) error {
	return errors.New("not implemented")
}

type ecdsaSignature struct {
	R *big.Int
	S *big.Int
}

// transformEcdsaSignature converts from an ASN.1 DER-encoded object to a fixed-length padded form which JWT
// implementations expect
func transformEcdsaSignature(length int64) func([]byte) ([]byte, error) {
	return func(rawSignature []byte) ([]byte, error) {
		var signature ecdsaSignature
		if rest, err := asn1.Unmarshal(rawSignature, &signature); err != nil {
			return nil, fmt.Errorf("failed to parse ECDSA signature: %w", err)
		} else if len(rest) != 0 {
			return nil, errors.New("found extra bytes after ECDSA signature")
		}

		result := make([]byte, 2*length)
		signature.R.FillBytes(result[0:length])
		signature.S.FillBytes(result[length:])

		return result, nil
	}
}
