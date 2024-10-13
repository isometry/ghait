package provider

import (
	"context"

	"errors"

	"github.com/golang-jwt/jwt/v4"
)

var ErrUnsupportedProvider = errors.New("unsupported provider")

type Provider interface {
	// Check checks the validity of the signer, returning an error if the signer
	// is invalid or misconfigured.
	Check() error

	// Sign signs the given claims and returns a JWT token string, as specified
	// by [jwt.Token.SignedString]
	Sign(claims jwt.Claims) (string, error)
}

type providerRegistry map[string]func(ctx context.Context, key string) (Provider, error)

var registry = providerRegistry{}

func Register(name string, newSigner func(ctx context.Context, key string) (Provider, error)) {
	registry[name] = newSigner
}

func Registered() []string {
	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	return keys
}

func NewSigner(ctx context.Context, provider, key string) (Provider, error) {
	newSigner, ok := registry[provider]
	if !ok {
		return nil, ErrUnsupportedProvider
	}

	return newSigner(ctx, key)
}
