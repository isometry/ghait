//go:build !ghait.no_file

package file

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestKey(t *testing.T) (*rsa.PrivateKey, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	return key, pemBytes
}

func TestNewSigner_FromFile(t *testing.T) {
	_, pemBytes := generateTestKey(t)

	keyFile := filepath.Join(t.TempDir(), "test.pem")
	require.NoError(t, os.WriteFile(keyFile, pemBytes, 0600))

	signer, err := NewSigner(context.Background(), keyFile)
	require.NoError(t, err)
	assert.NotNil(t, signer)
}

func TestNewSigner_FromString(t *testing.T) {
	_, pemBytes := generateTestKey(t)

	signer, err := NewSigner(context.Background(), string(pemBytes))
	require.NoError(t, err)
	assert.NotNil(t, signer)
}

func TestNewSigner_FromStringWithWhitespace(t *testing.T) {
	_, pemBytes := generateTestKey(t)

	// Simulate trailing newlines (e.g. from env vars or heredocs)
	keyWithWhitespace := string(pemBytes) + "\n\n  \t\n"
	signer, err := NewSigner(context.Background(), keyWithWhitespace)
	require.NoError(t, err)
	assert.NotNil(t, signer)
}

func TestNewSigner_InvalidPEM(t *testing.T) {
	signer, err := NewSigner(context.Background(), "not-a-pem-key")
	assert.Nil(t, signer)
	assert.EqualError(t, err, "failed to decode RSA private key")
}

func TestNewSigner_WrongPEMType(t *testing.T) {
	// Create an EC-style PEM block (wrong type for this provider)
	block := &pem.Block{Type: "EC PRIVATE KEY", Bytes: []byte("fake")}
	pemBytes := pem.EncodeToMemory(block)

	signer, err := NewSigner(context.Background(), string(pemBytes))
	assert.Nil(t, signer)
	assert.EqualError(t, err, "failed to decode RSA private key")
}

func TestNewSigner_EmptyKey(t *testing.T) {
	signer, err := NewSigner(context.Background(), "")
	assert.Nil(t, signer)
	assert.Error(t, err)
}

func TestNewSigner_WhitespaceOnlyKey(t *testing.T) {
	signer, err := NewSigner(context.Background(), "   \n\t  ")
	assert.Nil(t, signer)
	assert.Error(t, err)
}

func TestCheck(t *testing.T) {
	_, pemBytes := generateTestKey(t)

	signer, err := NewSigner(context.Background(), string(pemBytes))
	require.NoError(t, err)
	assert.NoError(t, signer.Check())
}

func TestSign_RoundTrip(t *testing.T) {
	key, pemBytes := generateTestKey(t)

	signer, err := NewSigner(context.Background(), string(pemBytes))
	require.NoError(t, err)

	claims := jwt.RegisteredClaims{
		Issuer:  "test-app",
		Subject: "test-subject",
	}

	tokenString, err := signer.Sign(claims)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	// Verify the token with the public key
	parsed, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		assert.Equal(t, jwt.SigningMethodRS256, token.Method)
		return &key.PublicKey, nil
	})
	require.NoError(t, err)
	assert.True(t, parsed.Valid)

	parsedClaims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	require.True(t, ok)
	assert.Equal(t, "test-app", parsedClaims.Issuer)
	assert.Equal(t, "test-subject", parsedClaims.Subject)
}
