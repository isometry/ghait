//go:build !no_aws

package ghait

import (
	_ "github.com/isometry/ghait/provider/aws"
)
