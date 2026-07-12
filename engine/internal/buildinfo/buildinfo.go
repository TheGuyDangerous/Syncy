// Package buildinfo exposes version and build metadata for the Syncy engine.
//
// The exported variables are overridden at build time via -ldflags; sensible
// defaults are used for local development builds so the binary always reports
// something meaningful.
package buildinfo

import (
	"fmt"
	"runtime"
)

// These variables are set at build time, for example:
//
//	go build -ldflags "-X github.com/TheGuyDangerous/Syncy/engine/internal/buildinfo.Version=v0.1.0"
var (
	// Version is the semantic version of the build.
	Version = "0.0.0-dev"
	// Commit is the git commit the build was produced from.
	Commit = "unknown"
	// Date is the build timestamp in RFC 3339 format.
	Date = "unknown"
)

// String returns a human-readable, single-line description of the build,
// including the Go runtime and target platform.
func String() string {
	return fmt.Sprintf("syncyd %s (commit %s, built %s, %s/%s, %s)",
		Version, Commit, Date, runtime.GOOS, runtime.GOARCH, runtime.Version())
}
