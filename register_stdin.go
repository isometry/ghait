//go:build !stdin

package ghait

import (
	// Register the stdin provider.
	_ "github.com/isometry/ghait/v80/provider/stdin"
)
