// Package provider offers functionality for token providers.
package provider

import (
	"context"
	"errors"
	"sync"

	"github.com/golang-jwt/jwt/v4"
)

// ErrUnsupportedProvider is returned when an unsupported provider is requested.
var ErrUnsupportedProvider = errors.New("unsupported provider")

// Provider is the interface that must be implemented by all token providers.
type Provider interface {
	// Check checks the validity of the signer, returning an error if the signer
	// is invalid or misconfigured.
	Check() error

	// Sign signs the given claims and returns a JWT token string, as specified
	// by [jwt.Token.SignedString]
	Sign(claims jwt.Claims) (string, error)
}

type providerRegistry map[string]func(ctx context.Context, key string) (Provider, error)

var (
	registry = providerRegistry{}
	mu       sync.RWMutex
)

// Register registers a new provider.
func Register(name string, newSigner func(ctx context.Context, key string) (Provider, error)) {
	mu.Lock()
	defer mu.Unlock()

	registry[name] = newSigner
}

// Registered returns a list of all registered providers.
func Registered() []string {
	mu.RLock()
	defer mu.RUnlock()

	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	return keys
}

// NewSigner creates a new signer for the given provider.
func NewSigner(ctx context.Context, provider, key string) (Provider, error) {
	mu.RLock()
	defer mu.RUnlock()

	newSigner, ok := registry[provider]
	if !ok {
		return nil, ErrUnsupportedProvider
	}

	return newSigner(ctx, key)
}
