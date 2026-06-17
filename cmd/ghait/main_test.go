//go:build !ghait.no_file

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isometry/ghait/v88"
)

// testKeyPEM returns a freshly generated RSA private key in PKCS#1 PEM form,
// suitable for the "file" provider.
func testKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}))
}

// tokenServer stands up an httptest server that answers the installation
// token-mint call, recording the request headers it observed.
func tokenServer(t *testing.T, seen *http.Header) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seen = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      "v1.test-installation-token",
			"expires_at": time.Now().Add(time.Hour).Format(time.RFC3339),
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newStatelessViper returns a fresh viper configured exactly like the CLI
// (GHAIT_-prefixed AutomaticEnv plus a bound "stateless" pflag) and applies the
// given command-line args, mirroring New()/initConfig() without touching the
// global viper singleton.
func newStatelessViper(t *testing.T, args ...string) *viper.Viper {
	t.Helper()
	v := viper.New()
	v.SetEnvPrefix("GHAIT")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.Bool("stateless", false, "")
	require.NoError(t, fs.Parse(args))
	require.NoError(t, v.BindPFlags(fs))
	return v
}

// TestStatelessOption exercises the full tri-state: the flag/env decision in
// statelessOption and the X-GitHub-Stateless-S2S-Token header it ultimately
// puts on the mint request. The header is checked end-to-end through NewGHAIT
// against a test server so a regression in either the decision or the value
// mapping is caught.
func TestStatelessOption(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		env        string // "" leaves GHAIT_STATELESS unset
		wantOption bool
		wantHeader string // expected header value on the wire; "" means absent
	}{
		{name: "omitted", wantOption: false, wantHeader: ""},
		{name: "bare flag", args: []string{"--stateless"}, wantOption: true, wantHeader: "enabled"},
		{name: "explicit true", args: []string{"--stateless=true"}, wantOption: true, wantHeader: "enabled"},
		{name: "explicit false", args: []string{"--stateless=false"}, wantOption: true, wantHeader: "disabled"},
		{name: "env true", env: "true", wantOption: true, wantHeader: "enabled"},
		{name: "env false", env: "false", wantOption: true, wantHeader: "disabled"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv("GHAIT_STATELESS", tc.env)
			}
			v := newStatelessViper(t, tc.args...)

			opt := statelessOption(v)
			if tc.wantOption {
				require.NotNil(t, opt, "expected an option when stateless is explicitly set")
			} else {
				assert.Nil(t, opt, "expected no option when stateless is omitted")
			}

			var seen http.Header
			srv := tokenServer(t, &seen)

			opts := []ghait.Option{ghait.WithURLs(srv.URL, "")}
			if opt != nil {
				opts = append(opts, opt)
			}

			cfg := ghait.NewConfig(12345, 67890, "file", testKeyPEM(t))
			factory, err := ghait.NewGHAIT(t.Context(), cfg, opts...)
			require.NoError(t, err)

			tok, err := factory.NewToken(t.Context())
			require.NoError(t, err)
			require.NotNil(t, tok)

			assert.Equal(t, tc.wantHeader, seen.Get("X-GitHub-Stateless-S2S-Token"))
			assert.Contains(t, seen.Get("Authorization"), "Bearer ",
				"ghinstallation app auth must still run beneath the option")
		})
	}
}
