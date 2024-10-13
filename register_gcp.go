//go:build !no_gcp

package ghait

import (
	"github.com/isometry/ghait/provider"
	"github.com/isometry/ghait/provider/gcp"
)

func init() {
	provider.Register("gcp", gcp.NewSigner)
}
