package aws

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/golang-jwt/jwt/v4"

	"github.com/isometry/ghait/provider"
)

func init() {
	provider.Register("aws", NewSigner)
}

// awsSigner implements provider.Provider & ghinstallation.Signer for AWS KMS.
type awsSigner struct {
	context context.Context
	client  *kms.Client
	key     string
}

// NewAwsSigner creates a new AWS signer.
func NewAwsSigner(ctx context.Context, key string, optFns ...func(*config.LoadOptions) error) (provider.Provider, error) {
	config, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, err
	}

	client := kms.NewFromConfig(config)

	return &awsSigner{
		context: ctx,
		client:  client,
		key:     key,
	}, nil
}

// NewSigner returns a new AWS signer with default configuration.
func NewSigner(ctx context.Context, key string) (provider.Provider, error) {
	return NewAwsSigner(ctx, key)
}

func (s *awsSigner) Check() error {
	input := &kms.DescribeKeyInput{
		KeyId: &s.key,
	}
	key, err := s.client.DescribeKey(s.context, input)
	if err != nil {
		return fmt.Errorf("failed to describe privateKey: %w", err)
	}

	if key.KeyMetadata.KeyState != types.KeyStateEnabled {
		return errors.New("privateKey is not enabled")
	}

	if key.KeyMetadata.KeyUsage != types.KeyUsageTypeSignVerify {
		return errors.New("privateKey is not for signing")
	}

	if !slices.Contains[[]types.SigningAlgorithmSpec](key.KeyMetadata.SigningAlgorithms, types.SigningAlgorithmSpecRsassaPkcs1V15Sha256) {
		return errors.New("privateKey does not support RS256 compatible signing algorithm")
	}

	return nil
}

// Sign signs the JWT claims with the RSA key.
func (s *awsSigner) Sign(claims jwt.Claims) (string, error) {
	method := &awsSigningMethod{
		context: s.context,
		client:  s.client,
	}
	return jwt.NewWithClaims(method, claims).SignedString(s.key)
}

// awsSigningMethod implements jwt.SigningMethod for AWS KMS.
type awsSigningMethod struct {
	context context.Context
	client  *kms.Client
}

func (s *awsSigningMethod) Alg() string {
	return "RS256"
}

func (s *awsSigningMethod) Sign(data string, ikey any) (string, error) {
	key, ok := ikey.(string)
	if !ok {
		return "", fmt.Errorf("invalid key reference type: %T", ikey)
	}

	input := kms.SignInput{
		KeyId:            &key,
		Message:          []byte(data),
		SigningAlgorithm: types.SigningAlgorithmSpecRsassaPkcs1V15Sha256,
	}
	output, err := s.client.Sign(s.context, &input)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(output.Signature), nil
}

func (s *awsSigningMethod) Verify(string, string, any) error {
	return errors.New("not implemented")
}
