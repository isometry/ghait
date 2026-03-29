//go:build !ghait.no_file

package ghait

import _ "github.com/isometry/ghait/provider/file" // Register the file provider by default.
