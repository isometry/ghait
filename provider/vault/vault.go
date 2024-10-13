package vault

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	vault "github.com/hashicorp/vault/api"

	"github.com/isometry/ghait/provider"
)

type vaultSigner struct {
	context context.Context
	client  *vault.Client
	key     string
}

func NewSigner(ctx context.Context, key string) (provider.Provider, error) {
	config := vault.DefaultConfig()
	if err := config.ReadEnvironment(); err != nil {
		return nil, err
	}

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &vaultSigner{
		context: ctx,
		client:  client,
		key:     key,
	}, nil
}

func (s *vaultSigner) Check() error {
	// TODO: implement appropriate checks
	return nil
}

// Sign signs the JWT claims with the RSA key.
func (s *vaultSigner) Sign(claims jwt.Claims) (string, error) {
	method := &vaultSigningMethod{
		context: s.context,
		client:  s.client,
	}
	return jwt.NewWithClaims(method, claims).SignedString(s.key)
}

type vaultSigningMethod struct {
	context context.Context
	client  *vault.Client
}

func (s *vaultSigningMethod) Alg() string {
	return "RS256"
}

func (s *vaultSigningMethod) Sign(data string, ikey any) (string, error) {
	key, ok := ikey.(string)
	if !ok {
		return "", fmt.Errorf("invalid key reference type: %T", ikey)
	}

	// key expected in the format "<transitPath>/sign/<keyName>",
	// but accept "<transitPath>/<keyName>" for convenience
	signPath := key
	if !strings.Contains(signPath, "/sign/") {
		transitPath, keyName := splitOnLast(key, "/")
		if keyName == "" {
			return "", errors.New("invalid key reference format: expected transitPath/keyName")
		}

		signPath = fmt.Sprintf("%s/sign/%s", transitPath, keyName)
	}

	encodedData := base64.StdEncoding.EncodeToString([]byte(data))

	input := map[string]any{
		"input":                encodedData,
		"hash_algorithm":       "sha2-256",
		"signature_algorithm":  "pkcs1v15",
		"marshaling_algorithm": "jws",
	}
	resp, err := s.client.Logical().WriteWithContext(s.context, signPath, input)
	if err != nil {
		return "", fmt.Errorf("failed to write to Vault: %w", err)
	}

	vaultSignature, ok := resp.Data["signature"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected signature type: %T", resp.Data["signature"])
	}

	return strings.TrimPrefix(vaultSignature, "vault:v1:"), nil
}

func (s *vaultSigningMethod) Verify(string, string, any) error {
	return errors.New("not implemented")
}

func splitOnLast(s, sep string) (string, string) {
	index := strings.LastIndex(s, sep)
	if index == -1 {
		return s, ""
	}
	return s[:index], s[index+len(sep):]
}
