//go:build ghait.no_file

package ghait

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/isometry/ghait/v84/provider"
)

func TestFileProviderNotRegistered(t *testing.T) {
	registered := provider.Registered()
	assert.False(t, slices.Contains(registered, "file"),
		"file provider should not be registered with ghait.no_file tag, got: %v", registered)
}
