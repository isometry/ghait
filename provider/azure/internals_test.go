//go:build !ghait.no_azure

package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseKeyID(t *testing.T) {
	tests := []struct {
		name       string
		keyID      string
		wantURL    string
		wantName   string
		wantVer    string
		wantErrMsg string
	}{
		{
			name:     "valid with version",
			keyID:    "https://myvault.vault.azure.net/keys/mykey/abc123",
			wantURL:  "https://myvault.vault.azure.net",
			wantName: "mykey",
			wantVer:  "abc123",
		},
		{
			name:     "valid without version",
			keyID:    "https://myvault.vault.azure.net/keys/mykey",
			wantURL:  "https://myvault.vault.azure.net",
			wantName: "mykey",
			wantVer:  "",
		},
		{
			name:       "wrong scheme",
			keyID:      "http://myvault.vault.azure.net/keys/mykey",
			wantErrMsg: "scheme must be https",
		},
		{
			name:       "missing host",
			keyID:      "https:///keys/mykey",
			wantErrMsg: "missing vault hostname",
		},
		{
			name:       "wrong path prefix",
			keyID:      "https://myvault.vault.azure.net/secrets/mysecret",
			wantErrMsg: "path must be /keys/<key-name>[/<key-version>]",
		},
		{
			name:       "path too short",
			keyID:      "https://myvault.vault.azure.net/keys",
			wantErrMsg: "path must be /keys/<key-name>[/<key-version>]",
		},
		{
			name:       "empty path",
			keyID:      "https://myvault.vault.azure.net",
			wantErrMsg: "path must be /keys/<key-name>[/<key-version>]",
		},
		{
			name:       "no scheme",
			keyID:      "myvault.vault.azure.net/keys/mykey",
			wantErrMsg: "scheme must be https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vaultURL, keyName, keyVersion, err := parseKeyID(tt.keyID)
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, vaultURL)
			assert.Equal(t, tt.wantName, keyName)
			assert.Equal(t, tt.wantVer, keyVersion)
		})
	}
}
