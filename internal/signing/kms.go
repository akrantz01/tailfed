package signing

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"

	"github.com/akrantz01/tailfed/internal/oidc"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/sirupsen/logrus"
)

// kmsBackend signs tokens using a key stored in AWS KMS
type kmsBackend struct {
	logger logrus.FieldLogger

	id        string
	arn       *string
	algorithm jose.SignatureAlgorithm
	client    *kms.Client
	signer    jose.Signer
}

var _ Backend = (*kmsBackend)(nil)

// NewKMS creates a new AWS KMS-backed signer
func NewKMS(logger logrus.FieldLogger, config aws.Config, alias string) (Backend, error) {
	client := kms.NewFromConfig(config)

	logger.WithField("key", alias).Debug("resolving KMS key...")
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

	logger.WithField("arn", *metadata.Arn).Debug("resolved and validated key")

	key, algorithm, err := newKmsKey(logger, metadata, client)
	if err != nil {
		return nil, err
	}
	logger.WithField("algorithm", algorithm).Debug("determined signing algorithm for key")

	signer, err := newKey(*metadata.KeyId, key, algorithm)
	if err != nil {
		return nil, err
	}

	logger.WithField("key", *metadata.Arn).Info("created new KMS signer")
	return &kmsBackend{logger, *metadata.KeyId, metadata.Arn, algorithm, client, signer}, nil
}

func (k *kmsBackend) Sign(claims oidc.Claims) (string, error) {
	return jwt.Signed(k.signer).Claims(claims).Serialize()
}

func (k *kmsBackend) PublicKey() (jose.JSONWebKey, error) {
	k.logger.Debug("getting public key")
	output, err := k.client.GetPublicKey(context.Background(), &kms.GetPublicKeyInput{KeyId: k.arn})
	if err != nil {
		return jose.JSONWebKey{}, err
	}

	k.logger.Debug("parsing encoded public key")
	parsed, err := x509.ParsePKIXPublicKey(output.PublicKey)
	if err != nil {
		return jose.JSONWebKey{}, fmt.Errorf("failed to parse DER encoded public key: %w", err)
	}

	public, ok := parsed.(crypto.PublicKey)
	if !ok {
		// this should never happen, but just in case
		return jose.JSONWebKey{}, errors.New("unknown public key type")
	}

	return jose.JSONWebKey{
		Use:       "sig",
		KeyID:     k.id,
		Key:       public,
		Algorithm: string(k.algorithm),
	}, nil
}

type kmsKey struct {
	logger logrus.FieldLogger
	client *kms.Client

	arn         *string
	algorithm   types.SigningAlgorithmSpec
	transformer func([]byte) ([]byte, error)
}

var _ jose.OpaqueSigner = (*kmsKey)(nil)

func newKmsKey(logger logrus.FieldLogger, metadata *types.KeyMetadata, client *kms.Client) (*kmsKey, jose.SignatureAlgorithm, error) {
	switch metadata.KeySpec {
	case types.KeySpecRsa2048:
		return &kmsKey{logger, client, metadata.Arn, types.SigningAlgorithmSpecRsassaPkcs1V15Sha256, nil}, jose.RS256, nil
	case types.KeySpecRsa3072:
		return &kmsKey{logger, client, metadata.Arn, types.SigningAlgorithmSpecRsassaPkcs1V15Sha384, nil}, jose.RS384, nil
	case types.KeySpecRsa4096:
		return &kmsKey{logger, client, metadata.Arn, types.SigningAlgorithmSpecRsassaPkcs1V15Sha512, nil}, jose.RS512, nil
	case types.KeySpecEccNistP256:
		return &kmsKey{logger, client, metadata.Arn, types.SigningAlgorithmSpecEcdsaSha256, transformEcdsaSignature(32)}, jose.ES256, nil
	case types.KeySpecEccNistP384:
		return &kmsKey{logger, client, metadata.Arn, types.SigningAlgorithmSpecEcdsaSha384, transformEcdsaSignature(48)}, jose.ES384, nil
	case types.KeySpecEccNistP521:
		return &kmsKey{logger, client, metadata.Arn, types.SigningAlgorithmSpecEcdsaSha512, transformEcdsaSignature(66)}, jose.ES512, nil
	default:
		return nil, "", fmt.Errorf("unsupported key type %q", metadata.KeySpec)
	}
}

func (k *kmsKey) Public() *jose.JSONWebKey {
	// Intentionally left unimplemented
	k.logger.Warn("unexpected call to kmsKey.Public, method is unimplemented")
	return nil
}

func (k *kmsKey) Algs() []jose.SignatureAlgorithm {
	return []jose.SignatureAlgorithm{jose.RS256, jose.RS384, jose.RS512, jose.ES256, jose.ES384, jose.ES512}
}

func (k *kmsKey) SignPayload(payload []byte, _ jose.SignatureAlgorithm) ([]byte, error) {
	k.logger.Debug("requesting payload signature")

	output, err := k.client.Sign(context.Background(), &kms.SignInput{
		KeyId:            k.arn,
		Message:          payload,
		MessageType:      types.MessageTypeRaw,
		SigningAlgorithm: k.algorithm,
	})
	if err != nil {
		return nil, err
	}

	signature := output.Signature
	if k.transformer != nil {
		k.logger.Debug("transforming signature to jws-compatible format")

		var err error
		if signature, err = k.transformer(signature); err != nil {
			return nil, fmt.Errorf("failed to transform signature to JWT-compatible format: %w", err)
		}
	}

	return signature, nil
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
