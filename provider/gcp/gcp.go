// Package gcp provides a Google Cloud Platform (GCP) KMS signer implementation.
package gcp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/api/option"

	kms "cloud.google.com/go/kms/apiv1"

	"github.com/isometry/ghait/provider"
)

func init() {
	provider.Register("gcp", NewSigner)
}

// gcpSigner implements provider.Provider & ghinstallation.Signer for GCP KMS.
type gcpSigner struct {
	context context.Context
	client  *kms.KeyManagementClient
	key     string
}

// NewGcpSigner creates a new GCP signer.
func NewGcpSigner(ctx context.Context, key string, opts ...option.ClientOption) (provider.Provider, error) {
	client, err := kms.NewKeyManagementClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &gcpSigner{
		context: ctx,
		client:  client,
		key:     key,
	}, nil
}

// NewSigner returns a new GCP signer with default configuration.
func NewSigner(ctx context.Context, key string) (provider.Provider, error) {
	return NewGcpSigner(ctx, key)
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

// gcpSigningMethod implements jwt.SigningMethod for GCP KMS.
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

	return base64.RawURLEncoding.EncodeToString(resp.GetSignature()), nil
}

func (s *gcpSigningMethod) Verify(string, string, any) error {
	return errors.New("not implemented")
}
