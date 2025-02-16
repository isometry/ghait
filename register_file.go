//go:build !no_file

package ghait

import (
	// Register the file provider.
	_ "github.com/isometry/ghait/provider/file"
)
