package signing

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/go-jose/go-jose/v4"
	"github.com/sirupsen/logrus"
)

// NewAWSKMS creates a new AWS KMS-backed signer
func NewAWSKMS(config aws.Config, references []string) (Backend, error) {
	logger := logrus.WithField("component", "signer").WithField("signer.backend", "aws-kms")

	client := kms.NewFromConfig(config)

	keys := []genericKey{}
	for _, ref := range references {
		key, err := resolveKmsKey(logger, client, ref)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve key %q: %w", ref, err)
		}

		keys = append(keys, key)
	}

	logger.WithField("signer.keys", len(keys)).Info("created new KMS signer")
	return generic(keys), nil
}

func resolveKmsKey(logger logrus.FieldLogger, client *kms.Client, alias string) (genericKey, error) {
	logger = logger.WithField("signer.key", alias)

	logger.Debug("resolving KMS key...")
	details, err := client.DescribeKey(context.Background(), &kms.DescribeKeyInput{KeyId: aws.String(alias)})
	if err != nil {
		return genericKey{}, fmt.Errorf("failed to describe key: %w", err)
	}
	metadata := details.KeyMetadata

	if !metadata.Enabled {
		return genericKey{}, errors.New("key is not enabled")
	}
	if metadata.KeyUsage != types.KeyUsageTypeSignVerify {
		return genericKey{}, errors.New("key is not configured for sign/verify")
	}

	logger = logger.WithField("signer.key", *metadata.Arn)
	logger.Debug("resolved and validated key")

	public, err := getKmsPublicKey(logger, client, metadata.Arn)
	if err != nil {
		return genericKey{}, err
	}

	key, algorithm, err := newKmsKey(logger, client, metadata)
	if err != nil {
		return genericKey{}, err
	}
	logger.WithField("signer.algorithm", algorithm).Debug("determined signing algorithm for key")

	signer, err := newKey(*metadata.KeyId, key, algorithm)
	if err != nil {
		return genericKey{}, err
	}

	return genericKey{
		algorithm: algorithm,
		signer:    signer,
		publicKey: jose.JSONWebKey{
			Use:       "sig",
			KeyID:     *metadata.KeyId,
			Key:       public,
			Algorithm: string(toJose[algorithm]),
		},
	}, nil
}

func getKmsPublicKey(logger logrus.FieldLogger, client *kms.Client, arn *string) (crypto.PublicKey, error) {
	logger.Debug("getting public key")
	output, err := client.GetPublicKey(context.Background(), &kms.GetPublicKeyInput{KeyId: arn})
	if err != nil {
		return nil, err
	}

	logger.Debug("parsing encoded public key")
	parsed, err := x509.ParsePKIXPublicKey(output.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DER encoded public key: %w", err)
	}

	public, ok := parsed.(crypto.PublicKey)
	if !ok {
		// this should never happen, but just in case
		return nil, errors.New("unknown public key type")
	}

	return public, nil
}

type kmsKey struct {
	logger logrus.FieldLogger
	client *kms.Client

	arn              *string
	signingAlgorithm types.SigningAlgorithmSpec
	joseAlgorithm    jose.SignatureAlgorithm
	transformer      func([]byte) ([]byte, error)
}

var _ jose.OpaqueSigner = (*kmsKey)(nil)

func newKmsKey(logger logrus.FieldLogger, client *kms.Client, metadata *types.KeyMetadata) (*kmsKey, Algorithm, error) {
	jwtAlgorithm, awsAlgorithm, transformer, err := kmsKeyAlgorithm(metadata.KeySpec)
	if err != nil {
		return nil, "", err
	}

	return &kmsKey{
		logger:           logger,
		client:           client,
		arn:              metadata.Arn,
		signingAlgorithm: awsAlgorithm,
		transformer:      transformer,
		joseAlgorithm:    toJose[jwtAlgorithm],
	}, jwtAlgorithm, nil

}

func kmsKeyAlgorithm(spec types.KeySpec) (Algorithm, types.SigningAlgorithmSpec, func([]byte) ([]byte, error), error) {
	switch spec {
	case types.KeySpecRsa2048:
		return RS256, types.SigningAlgorithmSpecRsassaPkcs1V15Sha256, nil, nil
	case types.KeySpecRsa3072:
		return RS384, types.SigningAlgorithmSpecRsassaPkcs1V15Sha384, nil, nil
	case types.KeySpecRsa4096:
		return RS512, types.SigningAlgorithmSpecRsassaPkcs1V15Sha512, nil, nil
	case types.KeySpecEccNistP256:
		return ES256, types.SigningAlgorithmSpecEcdsaSha256, transformEcdsaSignature(32), nil
	case types.KeySpecEccNistP384:
		return ES384, types.SigningAlgorithmSpecEcdsaSha384, transformEcdsaSignature(48), nil
	case types.KeySpecEccNistP521:
		return ES512, types.SigningAlgorithmSpecEcdsaSha512, transformEcdsaSignature(66), nil
	default:
		return "", "", nil, fmt.Errorf("unsupported key type %q", spec)
	}
}

func (k *kmsKey) Public() *jose.JSONWebKey {
	return nil
}

func (k *kmsKey) Algs() []jose.SignatureAlgorithm {
	return []jose.SignatureAlgorithm{k.joseAlgorithm}
}

func (k *kmsKey) SignPayload(payload []byte, _ jose.SignatureAlgorithm) ([]byte, error) {
	k.logger.Debug("requesting payload signature")

	output, err := k.client.Sign(context.Background(), &kms.SignInput{
		KeyId:            k.arn,
		Message:          payload,
		MessageType:      types.MessageTypeRaw,
		SigningAlgorithm: k.signingAlgorithm,
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
