//go:build !no_vault

package ghait

import (
	_ "github.com/isometry/ghait/provider/vault"
)
