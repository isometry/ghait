//go:build !no_aws

package ghait

import (
	"github.com/isometry/ghait/provider"
	"github.com/isometry/ghait/provider/aws"
)

func init() {
	provider.Register("aws", aws.NewSigner)
}
