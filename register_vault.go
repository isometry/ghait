//go:build !no_vault

package ghait

import (
	// Register the Vault provider.
	_ "github.com/isometry/ghait/v84/provider/vault"
)
