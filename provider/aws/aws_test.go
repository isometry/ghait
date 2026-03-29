//go:build !ghait.no_aws

package aws_test

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	awsprovider "github.com/isometry/ghait/v84/provider/aws"
)

// mockKMSClient implements the aws.KMSClient interface for testing.
type mockKMSClient struct {
	SignFunc        func(ctx context.Context, params *kms.SignInput, optFns ...func(*kms.Options)) (*kms.SignOutput, error)
	DescribeKeyFunc func(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error)
}

func (m *mockKMSClient) Sign(ctx context.Context, params *kms.SignInput, optFns ...func(*kms.Options)) (*kms.SignOutput, error) {
	return m.SignFunc(ctx, params, optFns...)
}

func (m *mockKMSClient) DescribeKey(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error) {
	return m.DescribeKeyFunc(ctx, params, optFns...)
}

func validKeyMetadata() *types.KeyMetadata {
	return &types.KeyMetadata{
		KeyState:          types.KeyStateEnabled,
		KeyUsage:          types.KeyUsageTypeSignVerify,
		KeySpec:           types.KeySpecRsa2048,
		SigningAlgorithms: []types.SigningAlgorithmSpec{types.SigningAlgorithmSpecRsassaPkcs1V15Sha256},
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name       string
		metadata   *types.KeyMetadata
		descErr    error
		wantErrMsg string
	}{
		{name: "success", metadata: validKeyMetadata()},
		{name: "describe key error", descErr: errors.New("access denied"), wantErrMsg: "failed to describe privateKey"},
		{
			name: "key not enabled",
			metadata: func() *types.KeyMetadata {
				m := validKeyMetadata()
				m.KeyState = types.KeyStateDisabled
				return m
			}(),
			wantErrMsg: "not enabled",
		},
		{
			name: "wrong key usage",
			metadata: func() *types.KeyMetadata {
				m := validKeyMetadata()
				m.KeyUsage = types.KeyUsageTypeEncryptDecrypt
				return m
			}(),
			wantErrMsg: "not for signing",
		},
		{
			name: "missing signing algorithm",
			metadata: func() *types.KeyMetadata {
				m := validKeyMetadata()
				m.SigningAlgorithms = []types.SigningAlgorithmSpec{types.SigningAlgorithmSpecRsassaPssSha256}
				return m
			}(),
			wantErrMsg: "RS256",
		},
		{
			name: "wrong key spec",
			metadata: func() *types.KeyMetadata {
				m := validKeyMetadata()
				m.KeySpec = types.KeySpecRsa4096
				return m
			}(),
			wantErrMsg: "RSA 2048",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockKMSClient{
				DescribeKeyFunc: func(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error) {
					if tt.descErr != nil {
						return nil, tt.descErr
					}
					return &kms.DescribeKeyOutput{KeyMetadata: tt.metadata}, nil
				},
			}
			s := awsprovider.NewAwsSignerFromClient(t.Context(), "test-key", mock)
			err := s.Check()
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSign_Success(t *testing.T) {
	fakeSignature := []byte("fake-signature-bytes")

	mock := &mockKMSClient{
		SignFunc: func(ctx context.Context, params *kms.SignInput, optFns ...func(*kms.Options)) (*kms.SignOutput, error) {
			assert.Equal(t, types.SigningAlgorithmSpecRsassaPkcs1V15Sha256, params.SigningAlgorithm)
			assert.NotEmpty(t, params.Message)
			return &kms.SignOutput{Signature: fakeSignature}, nil
		},
	}

	s := awsprovider.NewAwsSignerFromClient(t.Context(), "test-key-id", mock)

	claims := jwt.RegisteredClaims{Issuer: "test-app"}
	token, err := s.Sign(claims)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify the token has 3 parts (header.payload.signature)
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)

	// Verify the signature part matches our fake signature
	assert.Equal(t, base64.RawURLEncoding.EncodeToString(fakeSignature), parts[2])
}

func TestSign_KMSError(t *testing.T) {
	mock := &mockKMSClient{
		SignFunc: func(ctx context.Context, params *kms.SignInput, optFns ...func(*kms.Options)) (*kms.SignOutput, error) {
			return nil, errors.New("throttling exception")
		},
	}

	s := awsprovider.NewAwsSignerFromClient(t.Context(), "test-key-id", mock)

	claims := jwt.RegisteredClaims{Issuer: "test-app"}
	_, err := s.Sign(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "throttling exception")
}
