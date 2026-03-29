//go:build !ghait.no_gcp

package gcp_test

import (
	"context"
	"encoding/base64"
	"net"
	"strings"
	"testing"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/isometry/ghait/v84/provider/gcp"
)

// fakeKMSServer implements the KeyManagementServiceServer interface for testing.
type fakeKMSServer struct {
	kmspb.UnimplementedKeyManagementServiceServer
	asymmetricSignFunc      func(ctx context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error)
	getCryptoKeyVersionFunc func(ctx context.Context, req *kmspb.GetCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error)
}

func (f *fakeKMSServer) AsymmetricSign(ctx context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
	if f.asymmetricSignFunc != nil {
		return f.asymmetricSignFunc(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "not configured")
}

func (f *fakeKMSServer) GetCryptoKeyVersion(ctx context.Context, req *kmspb.GetCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	if f.getCryptoKeyVersionFunc != nil {
		return f.getCryptoKeyVersionFunc(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "not configured")
}

// startFakeKMSServer starts a gRPC server with the fake KMS service and returns
// client options to connect to it. The server is stopped when the test completes.
func startFakeKMSServer(t *testing.T, srv *fakeKMSServer) []option.ClientOption {
	t.Helper()

	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	gsrv := grpc.NewServer()
	kmspb.RegisterKeyManagementServiceServer(gsrv, srv)
	t.Cleanup(gsrv.Stop)

	go gsrv.Serve(l) //nolint:errcheck // Stopped by t.Cleanup

	return []option.ClientOption{
		option.WithEndpoint(l.Addr().String()),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}
}

func TestCheck(t *testing.T) {
	keyName := "projects/test/locations/global/keyRings/ring/cryptoKeys/key/versions/1"

	tests := []struct {
		name       string
		state      kmspb.CryptoKeyVersion_CryptoKeyVersionState
		algorithm  kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm
		rpcErr     error
		wantErrMsg string
	}{
		{name: "success", state: kmspb.CryptoKeyVersion_ENABLED, algorithm: kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_2048_SHA256},
		{name: "key not enabled", state: kmspb.CryptoKeyVersion_DISABLED, algorithm: kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_2048_SHA256, wantErrMsg: "not enabled"},
		{name: "wrong algorithm", state: kmspb.CryptoKeyVersion_ENABLED, algorithm: kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_4096_SHA256, wantErrMsg: "RS256"},
		{name: "get key error", rpcErr: status.Error(codes.NotFound, "key not found"), wantErrMsg: "failed to get crypto key version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &fakeKMSServer{
				getCryptoKeyVersionFunc: func(ctx context.Context, req *kmspb.GetCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
					if tt.rpcErr != nil {
						return nil, tt.rpcErr
					}
					return &kmspb.CryptoKeyVersion{State: tt.state, Algorithm: tt.algorithm}, nil
				},
			}
			opts := startFakeKMSServer(t, srv)
			signer, err := gcp.NewGcpSigner(t.Context(), keyName, opts...)
			require.NoError(t, err)
			err = signer.Check()
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
	fakeSignature := []byte("fake-gcp-signature")
	keyName := "projects/test/locations/global/keyRings/ring/cryptoKeys/key/versions/1"

	srv := &fakeKMSServer{
		asymmetricSignFunc: func(ctx context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
			assert.Equal(t, keyName, req.Name)
			assert.NotEmpty(t, req.Data)
			return &kmspb.AsymmetricSignResponse{Signature: fakeSignature}, nil
		},
	}

	opts := startFakeKMSServer(t, srv)
	signer, err := gcp.NewGcpSigner(t.Context(), keyName, opts...)
	require.NoError(t, err)

	claims := jwt.RegisteredClaims{Issuer: "test-app"}
	token, err := signer.Sign(claims)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify the token has 3 parts and the signature matches
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	assert.Equal(t, base64.RawURLEncoding.EncodeToString(fakeSignature), parts[2])
}

func TestSign_KMSError(t *testing.T) {
	srv := &fakeKMSServer{
		asymmetricSignFunc: func(ctx context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
			return nil, status.Error(codes.PermissionDenied, "permission denied")
		},
	}

	opts := startFakeKMSServer(t, srv)
	keyName := "projects/test/locations/global/keyRings/ring/cryptoKeys/key/versions/1"
	signer, err := gcp.NewGcpSigner(t.Context(), keyName, opts...)
	require.NoError(t, err)

	claims := jwt.RegisteredClaims{Issuer: "test-app"}
	_, err = signer.Sign(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PermissionDenied")
}
