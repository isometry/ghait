//go:build !ghait.no_vault

package vault_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isometry/ghait/v84/provider"
	"github.com/isometry/ghait/v84/provider/vault"
)

// newTestVaultSigner creates a provider.Provider pointed at a test HTTP server.
func newTestVaultSigner(t *testing.T, handler http.Handler, key string) provider.Provider {
	t.Helper()

	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	signer, err := vault.NewVaultSigner(t.Context(), key, func(c *vaultapi.Config) {
		c.Address = ts.URL
	})
	require.NoError(t, err)

	return signer
}

func TestCheck(t *testing.T) {
	keyHandler := func(keyType string, supportsSigning bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/transit/keys/mykey", r.URL.Path)
			resp := map[string]any{
				"data": map[string]any{
					"type":             keyType,
					"supports_signing": supportsSigning,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}
	}

	tests := []struct {
		name       string
		key        string
		handler    http.HandlerFunc
		wantErrMsg string
	}{
		{name: "success", key: "transit/mykey", handler: keyHandler("rsa-2048", true)},
		{name: "explicit sign path", key: "transit/sign/mykey", handler: keyHandler("rsa-2048", true)},
		{name: "wrong key type", key: "transit/mykey", handler: keyHandler("ed25519", true), wantErrMsg: "not rsa-2048"},
		{name: "no signing support", key: "transit/mykey", handler: keyHandler("rsa-2048", false), wantErrMsg: "does not support signing"},
		{
			name: "key not found",
			key:  "transit/mykey",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}),
			wantErrMsg: "key not found",
		},
		{
			name: "invalid key format",
			key:  "noSlashKey",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("should not make HTTP request for invalid key format")
				http.Error(w, "unexpected request", http.StatusInternalServerError)
			}),
			wantErrMsg: "invalid key reference format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := newTestVaultSigner(t, tt.handler, tt.key)
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
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/transit/sign/mykey", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var input map[string]any
		require.NoError(t, json.Unmarshal(body, &input))
		assert.NotEmpty(t, input["input"])
		assert.Equal(t, "sha2-256", input["hash_algorithm"])
		assert.Equal(t, "pkcs1v15", input["signature_algorithm"])
		assert.Equal(t, "jws", input["marshaling_algorithm"])

		resp := map[string]any{
			"data": map[string]any{
				"signature": "vault:v1:dGVzdC1zaWduYXR1cmU",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	signer := newTestVaultSigner(t, handler, "transit/mykey")
	claims := jwt.RegisteredClaims{Issuer: "test-app"}
	token, err := signer.Sign(claims)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify JWT has 3 parts
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)

	// The signature should be the vault signature with prefix stripped
	assert.Equal(t, "dGVzdC1zaWduYXR1cmU", parts[2])
}

func TestSign_PathVariants(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/transit/sign/mykey", r.URL.Path)
		resp := map[string]any{
			"data": map[string]any{
				"signature": "vault:v1:c2lnbmF0dXJl",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	tests := []struct {
		name string
		key  string
	}{
		{name: "auto sign path injection", key: "transit/mykey"},
		{name: "explicit sign path", key: "transit/sign/mykey"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := newTestVaultSigner(t, handler, tt.key)
			claims := jwt.RegisteredClaims{Issuer: "test-app"}
			_, err := signer.Sign(claims)
			require.NoError(t, err)
		})
	}
}

func TestSign_VaultError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []string{"permission denied"},
		})
	})

	signer := newTestVaultSigner(t, handler, "transit/mykey")
	claims := jwt.RegisteredClaims{Issuer: "test-app"}
	_, err := signer.Sign(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write to Vault")
}
