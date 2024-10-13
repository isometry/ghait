package gcp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/golang-jwt/jwt/v4"

	"github.com/isometry/ghait/provider"
)

type gcpSigner struct {
	context context.Context
	client  *kms.KeyManagementClient
	key     string
}

func NewSigner(ctx context.Context, key string) (provider.Provider, error) {
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, err
	}

	return &gcpSigner{
		context: ctx,
		client:  client,
		key:     key,
	}, nil
}

func (s *gcpSigner) Check() error {
	// TODO: implement appropriate checks
	return nil
}

// Sign signs the JWT claims with the RSA key.
func (s *gcpSigner) Sign(claims jwt.Claims) (string, error) {
	method := &gcpSigningMethod{
		context: s.context,
		client:  s.client,
	}
	return jwt.NewWithClaims(method, claims).SignedString(s.key)
}

type gcpSigningMethod struct {
	context context.Context
	client  *kms.KeyManagementClient
}

func (s *gcpSigningMethod) Alg() string {
	return "RS256"
}

func (s *gcpSigningMethod) Sign(data string, ikey any) (string, error) {
	key, ok := ikey.(string)
	if !ok {
		return "", fmt.Errorf("invalid key reference type: %T", ikey)
	}

	req := &kmspb.AsymmetricSignRequest{
		Name: key,
		Data: []byte(data),
	}
	resp, err := s.client.AsymmetricSign(s.context, req)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(resp.Signature), nil
}

func (s *gcpSigningMethod) Verify(string, string, any) error {
	return errors.New("not implemented")
}
