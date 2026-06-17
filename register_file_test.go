//go:build !ghait.no_file

package ghait

import (
	"slices"
	"testing"

	"github.com/isometry/ghait/v88/provider"
	"github.com/stretchr/testify/assert"
)

func TestFileProviderRegistered(t *testing.T) {
	registered := provider.Registered()
	assert.True(t, slices.Contains(registered, "file"),
		"file provider should be registered by default, got: %v", registered)
}
