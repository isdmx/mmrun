// Package version exposes build metadata injected at link time via -ldflags.
package version

import (
	"fmt"
	"runtime"
)

// These variables are overridden at build time with:
//
//	-ldflags "-X github.com/isdmx/mmrun/internal/version.Version=v1.2.3 ..."
var (
	// Version is the semantic version (from the git tag on release builds).
	Version = "dev"
	// Commit is the short git commit hash.
	Commit = "none"
	// Date is the build timestamp (RFC3339, UTC).
	Date = "unknown"
)

// Info holds the resolved build metadata.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Get returns the current build metadata.
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String renders a one-line human-readable version summary.
func String() string {
	return fmt.Sprintf("mmrun %s (commit %s, built %s, %s %s/%s)",
		Version, Commit, Date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
