package file

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v4"

	"github.com/isometry/ghait/provider"
)

func init() {
	provider.Register("file", NewSigner)
}

type rsaSigner struct {
	context context.Context
	key     *rsa.PrivateKey
}

func NewSigner(ctx context.Context, key string) (provider.Provider, error) {
	var keyBytes []byte

	if _, err := os.Stat(key); err == nil {
		keyBytes, err = os.ReadFile(key)
		if err != nil {
			return nil, err
		}
	} else {
		keyBytes = []byte(key)
	}

	if keyBytes == nil {
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

	return &rsaSigner{
		context: ctx,
		key:     privateKey,
	}, nil
}

func (s *rsaSigner) Check() error {
	// validated within NewSigner
	return nil
}

// Sign signs the JWT claims with the RSA key.
func (s *rsaSigner) Sign(claims jwt.Claims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(s.key)
}
