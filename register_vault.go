//go:build !no_vault

package ghait

import (
	"github.com/isometry/ghait/provider"
	"github.com/isometry/ghait/provider/vault"
)

func init() {
	provider.Register("vault", vault.NewSigner)
}
