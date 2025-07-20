package provider_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/isometry/ghait/provider"
)

// MockProvider implements the Provider interface for testing
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) Check() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProvider) Sign(claims jwt.Claims) (string, error) {
	args := m.Called(claims)
	return args.String(0), args.Error(1)
}

func TestRegistered(t *testing.T) {
	testProvider1 := func(ctx context.Context, key string) (provider.Provider, error) {
		return &MockProvider{}, nil
	}
	testProvider2 := func(ctx context.Context, key string) (provider.Provider, error) {
		return &MockProvider{}, nil
	}

	provider.Register("test1", testProvider1)
	provider.Register("test2", testProvider2)

	registered := provider.Registered()

	assert.ElementsMatch(t, []string{"test1", "test2"}, registered)
}

func TestNewSigner_Success(t *testing.T) {
	expectedProvider := &MockProvider{}
	testProvider := func(ctx context.Context, key string) (provider.Provider, error) {
		assert.Equal(t, "test-key", key)
		return expectedProvider, nil
	}

	provider.Register("test", testProvider)

	ctx := context.Background()
	signer, err := provider.NewSigner(ctx, "test", "test-key")

	assert.NoError(t, err)
	assert.Equal(t, expectedProvider, signer)
}

func TestNewSigner_UnsupportedProvider(t *testing.T) {
	ctx := context.Background()
	signer, err := provider.NewSigner(ctx, "nonexistent", "test-key")

	assert.Nil(t, signer)
	assert.Equal(t, provider.ErrUnsupportedProvider, err)
}

func TestNewSigner_ProviderError(t *testing.T) {
	expectedError := errors.New("provider creation failed")
	testProvider := func(ctx context.Context, key string) (provider.Provider, error) {
		return nil, expectedError
	}

	provider.Register("test", testProvider)

	ctx := context.Background()
	signer, err := provider.NewSigner(ctx, "test", "test-key")

	assert.Nil(t, signer)
	assert.Equal(t, expectedError, err)
}
