// Package version exposes build metadata injected at link time via -ldflags,
// falling back to debug.ReadBuildInfo for go install builds.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
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
	v, c := Version, Commit
	if v == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				v = info.Main.Version
			}
			for _, s := range info.Settings {
				if s.Key == "vcs.revision" && c == "none" {
					c = s.Value
					if len(c) > 7 {
						c = c[:7]
					}
				}
			}
		}
	}
	return Info{
		Version:   v,
		Commit:    c,
		Date:      Date,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String renders a one-line human-readable version summary.
func String() string {
	i := Get()
	return fmt.Sprintf("mmrun %s (commit %s, built %s, %s %s/%s)",
		i.Version, i.Commit, Date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
