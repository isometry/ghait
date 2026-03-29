//go:build !ghait.no_azure

// Package azure provides an Azure Key Vault signer implementation.
package azure

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"github.com/golang-jwt/jwt/v4"

	"github.com/isometry/ghait/v84/provider"
)

func init() {
	provider.Register("azure", NewSigner)
}

// azureSigner implements provider.Provider & ghinstallation.Signer for Azure Key Vault.
type azureSigner struct {
	context    context.Context
	client     *azkeys.Client
	keyName    string
	keyVersion string
}

// NewAzureSigner creates a new Azure Key Vault signer.
func NewAzureSigner(ctx context.Context, vaultURL, keyName, keyVersion string, credential azcore.TokenCredential, opts *azkeys.ClientOptions) (provider.Provider, error) {
	client, err := azkeys.NewClient(vaultURL, credential, opts)
	if err != nil {
		return nil, err
	}

	return &azureSigner{
		context:    ctx,
		client:     client,
		keyName:    keyName,
		keyVersion: keyVersion,
	}, nil
}

// NewSigner returns a new Azure Key Vault signer with default configuration.
// The key parameter is the full key identifier URL:
// https://<vault-name>.vault.azure.net/keys/<key-name>[/<key-version>]
func NewSigner(ctx context.Context, key string) (provider.Provider, error) {
	vaultURL, keyName, keyVersion, err := parseKeyID(key)
	if err != nil {
		return nil, fmt.Errorf("invalid Azure Key Vault key identifier: %w", err)
	}

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	return NewAzureSigner(ctx, vaultURL, keyName, keyVersion, credential, nil)
}

func (s *azureSigner) Check() error {
	resp, err := s.client.GetKey(s.context, s.keyName, s.keyVersion, nil)
	if err != nil {
		return fmt.Errorf("failed to get key: %w", err)
	}

	if resp.Attributes == nil || resp.Attributes.Enabled == nil || !*resp.Attributes.Enabled {
		return errors.New("key is not enabled")
	}

	if resp.Key == nil {
		return errors.New("key metadata is missing")
	}

	if resp.Key.Kty == nil || (*resp.Key.Kty != azkeys.KeyTypeRSA && *resp.Key.Kty != azkeys.KeyTypeRSAHSM) {
		return errors.New("key is not an RSA key")
	}

	if len(resp.Key.N) != 256 { // 2048 bits / 8
		return errors.New("key is not RSA 2048")
	}

	hasSign := slices.ContainsFunc(resp.Key.KeyOps, func(op *azkeys.KeyOperation) bool {
		return op != nil && *op == azkeys.KeyOperationSign
	})
	if !hasSign {
		return errors.New("key does not support sign operation")
	}

	return nil
}

// Sign signs the JWT claims with the RSA key.
func (s *azureSigner) Sign(claims jwt.Claims) (string, error) {
	method := &azureSigningMethod{
		context:    s.context,
		client:     s.client,
		keyName:    s.keyName,
		keyVersion: s.keyVersion,
	}
	return jwt.NewWithClaims(method, claims).SignedString(s.keyName)
}

// azureSigningMethod implements jwt.SigningMethod for Azure Key Vault.
type azureSigningMethod struct {
	context    context.Context
	client     *azkeys.Client
	keyName    string
	keyVersion string
}

func (s *azureSigningMethod) Alg() string {
	return "RS256"
}

func (s *azureSigningMethod) Sign(data string, _ any) (string, error) {
	digest := sha256.Sum256([]byte(data))

	alg := azkeys.SignatureAlgorithmRS256
	params := azkeys.SignParameters{
		Algorithm: &alg,
		Value:     digest[:],
	}
	resp, err := s.client.Sign(s.context, s.keyName, s.keyVersion, params, nil)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(resp.Result), nil
}

func (s *azureSigningMethod) Verify(string, string, any) error {
	return errors.New("not implemented")
}

// parseKeyID parses an Azure Key Vault key identifier URL into its components.
// Expected format: https://<vault-name>.vault.azure.net/keys/<key-name>[/<key-version>]
func parseKeyID(keyID string) (vaultURL, keyName, keyVersion string, err error) {
	u, err := url.Parse(keyID)
	if err != nil {
		return "", "", "", err
	}

	if u.Scheme != "https" {
		return "", "", "", errors.New("scheme must be https")
	}

	if u.Host == "" {
		return "", "", "", errors.New("missing vault hostname")
	}

	// Path should be /keys/<name> or /keys/<name>/<version>
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "keys" {
		return "", "", "", errors.New("path must be /keys/<key-name>[/<key-version>]")
	}

	vaultURL = fmt.Sprintf("https://%s", u.Host)
	keyName = parts[1]
	if len(parts) >= 3 {
		keyVersion = parts[2]
	}

	return vaultURL, keyName, keyVersion, nil
}
