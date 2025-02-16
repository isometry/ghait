//go:build !no_gcp

package ghait

import (
	// Register the GCP provider.
	_ "github.com/isometry/ghait/provider/gcp"
)
