//go:build !no_aws

package ghait

import (
	// Register the AWS provider.
	_ "github.com/isometry/ghait/provider/aws"
)
