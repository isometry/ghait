//go:build ghait.no_file

package ghait

import (
	"slices"
	"testing"

	"github.com/isometry/ghait/v88/provider"
	"github.com/stretchr/testify/assert"
)

func TestFileProviderNotRegistered(t *testing.T) {
	registered := provider.Registered()
	assert.False(t, slices.Contains(registered, "file"),
		"file provider should not be registered with ghait.no_file tag, got: %v", registered)
}
