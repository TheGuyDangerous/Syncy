// Package buildinfo exposes version and build metadata, overridable via -ldflags.
package buildinfo

import (
	"fmt"
	"runtime"
)

var (
	Version = "0.0.0-dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func String() string {
	return fmt.Sprintf("syncyd %s (commit %s, built %s, %s/%s, %s)",
		Version, Commit, Date, runtime.GOOS, runtime.GOARCH, runtime.Version())
}
