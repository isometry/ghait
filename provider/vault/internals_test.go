//go:build !ghait.no_vault

package vault

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCutLast(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		sep        string
		wantBefore string
		wantAfter  string
		wantFound  bool
	}{
		{
			name:       "simple split",
			input:      "transit/sign/mykey",
			sep:        "/",
			wantBefore: "transit/sign",
			wantAfter:  "mykey",
			wantFound:  true,
		},
		{
			name:       "no separator",
			input:      "keyname",
			sep:        "/",
			wantBefore: "keyname",
			wantAfter:  "",
			wantFound:  false,
		},
		{
			name:       "single separator",
			input:      "transit/mykey",
			sep:        "/",
			wantBefore: "transit",
			wantAfter:  "mykey",
			wantFound:  true,
		},
		{
			name:       "multiple separators",
			input:      "a/b/c/d",
			sep:        "/",
			wantBefore: "a/b/c",
			wantAfter:  "d",
			wantFound:  true,
		},
		{
			name:       "empty string",
			input:      "",
			sep:        "/",
			wantBefore: "",
			wantAfter:  "",
			wantFound:  false,
		},
		{
			name:       "trailing separator",
			input:      "transit/",
			sep:        "/",
			wantBefore: "transit",
			wantAfter:  "",
			wantFound:  true,
		},
		{
			name:       "leading separator",
			input:      "/mykey",
			sep:        "/",
			wantBefore: "",
			wantAfter:  "mykey",
			wantFound:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before, after, found := cutLast(tt.input, tt.sep)
			assert.Equal(t, tt.wantBefore, before)
			assert.Equal(t, tt.wantAfter, after)
			assert.Equal(t, tt.wantFound, found)
		})
	}
}
