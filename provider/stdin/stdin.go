// Package stdin provides a stdin-based implementation of the ghait.Provider interface.
package stdin

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v4"

	"github.com/isometry/ghait/provider"
)

func init() {
	provider.Register("stdin", NewSigner)
}

// stdinSigner implements provider.Provider & ghinstallation.Signer with the RSA key retrieved from stdin.
type stdinSigner struct {
	context context.Context
	key     *rsa.PrivateKey
}

// NewSigner creates a new file signer.
func NewSigner(ctx context.Context, key string) (provider.Provider, error) {
	keyBytes := []byte(strings.TrimSpace(key))
	if len(keyBytes) == 0 {
		return nil, errors.New("empty key")
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("failed to decode RSA private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
	}

	return &stdinSigner{
		context: ctx,
		key:     privateKey,
	}, nil
}

func (s *stdinSigner) Check() error {
	// validated within NewSigner
	return nil
}

// Sign signs the JWT claims with the RSA key.
func (s *stdinSigner) Sign(claims jwt.Claims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(s.key)
}
