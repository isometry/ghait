//go:build !ghait.no_azure

package azure_test

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys/fake"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isometry/ghait/provider"
	"github.com/isometry/ghait/provider/azure"
)

// fakeCredential implements azcore.TokenCredential for testing.
type fakeCredential struct{}

func (f *fakeCredential) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "fake-token", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

// newTestAzureSigner creates a provider.Provider backed by a fake server.
func newTestAzureSigner(t *testing.T, srv fake.Server) provider.Provider {
	t.Helper()

	transport := fake.NewServerTransport(&srv)
	opts := &azkeys.ClientOptions{}
	opts.Transport = transport

	signer, err := azure.NewAzureSigner(
		t.Context(),
		"https://fakevault.vault.azure.net",
		"mykey",
		"v1",
		&fakeCredential{},
		opts,
	)
	require.NoError(t, err)

	return signer
}

func ptr[T any](v T) *T {
	return &v
}

func validGetKeyResponse() azkeys.GetKeyResponse {
	return azkeys.GetKeyResponse{
		KeyBundle: azkeys.KeyBundle{
			Attributes: &azkeys.KeyAttributes{
				Enabled: ptr(true),
			},
			Key: &azkeys.JSONWebKey{
				Kty:    ptr(azkeys.KeyTypeRSA),
				N:      make([]byte, 256), // 2048 bits / 8
				KeyOps: []*azkeys.KeyOperation{ptr(azkeys.KeyOperationSign)},
			},
		},
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name       string
		modifyResp func(*azkeys.GetKeyResponse)
		serverErr  string // if non-empty, fake returns this error instead of a response
		wantErrMsg string
	}{
		{name: "success"},
		{name: "key not enabled", modifyResp: func(r *azkeys.GetKeyResponse) { r.Attributes.Enabled = ptr(false) }, wantErrMsg: "not enabled"},
		{name: "not RSA key", modifyResp: func(r *azkeys.GetKeyResponse) { r.Key.Kty = ptr(azkeys.KeyTypeEC) }, wantErrMsg: "not an RSA key"},
		{name: "wrong key size", modifyResp: func(r *azkeys.GetKeyResponse) { r.Key.N = make([]byte, 512) }, wantErrMsg: "not RSA 2048"},
		{name: "RSA-HSM accepted", modifyResp: func(r *azkeys.GetKeyResponse) { r.Key.Kty = ptr(azkeys.KeyTypeRSAHSM) }},
		{name: "no sign operation", modifyResp: func(r *azkeys.GetKeyResponse) { r.Key.KeyOps = []*azkeys.KeyOperation{ptr(azkeys.KeyOperationEncrypt)} }, wantErrMsg: "does not support sign"},
		{name: "get key error", serverErr: "KeyNotFound", wantErrMsg: "failed to get key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := fake.Server{
				GetKey: func(ctx context.Context, name string, version string, options *azkeys.GetKeyOptions) (resp azfake.Responder[azkeys.GetKeyResponse], errResp azfake.ErrorResponder) {
					if tt.serverErr != "" {
						errResp.SetResponseError(404, tt.serverErr)
						return
					}
					r := validGetKeyResponse()
					if tt.modifyResp != nil {
						tt.modifyResp(&r)
					}
					resp.SetResponse(200, r, nil)
					return
				},
			}
			signer := newTestAzureSigner(t, srv)
			err := signer.Check()
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
	fakeSignature := []byte("fake-azure-signature")

	srv := fake.Server{
		Sign: func(ctx context.Context, name string, version string, parameters azkeys.SignParameters, options *azkeys.SignOptions) (resp azfake.Responder[azkeys.SignResponse], errResp azfake.ErrorResponder) {
			assert.Contains(t, name, "mykey")
			assert.Equal(t, ptr(azkeys.SignatureAlgorithmRS256), parameters.Algorithm)
			assert.Len(t, parameters.Value, 32) // SHA256 digest is 32 bytes

			resp.SetResponse(200, azkeys.SignResponse{
				KeyOperationResult: azkeys.KeyOperationResult{
					Result: fakeSignature,
				},
			}, nil)
			return
		},
	}

	signer := newTestAzureSigner(t, srv)
	claims := jwt.RegisteredClaims{Issuer: "test-app"}
	token, err := signer.Sign(claims)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify JWT has 3 parts and signature matches
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	assert.Equal(t, base64.RawURLEncoding.EncodeToString(fakeSignature), parts[2])
}

func TestSign_Error(t *testing.T) {
	srv := fake.Server{
		Sign: func(ctx context.Context, name string, version string, parameters azkeys.SignParameters, options *azkeys.SignOptions) (resp azfake.Responder[azkeys.SignResponse], errResp azfake.ErrorResponder) {
			errResp.SetResponseError(403, "Forbidden")
			return
		},
	}

	signer := newTestAzureSigner(t, srv)
	claims := jwt.RegisteredClaims{Issuer: "test-app"}
	_, err := signer.Sign(claims)
	require.Error(t, err)
}
