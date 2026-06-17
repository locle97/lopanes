// Package version exposes the build version of quash-board.
package version

import (
	"runtime/debug"
	"strings"
)

// fallback is reported for builds without a real release version, such as local
// `go build`/`go run`. Since Go 1.24 those builds carry a VCS pseudo-version
// (a "v0.0.0-<timestamp>-<commit>" string) rather than a tag, which we also
// treat as non-release.
const fallback = "dev"

// Version returns the release version this binary was built from. Binaries
// installed via `go install <module>@vX.Y.Z` report the tag (e.g. "v0.1.0");
// any local or untagged build reports "dev".
func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil {
		return resolve("", false)
	}
	return resolve(info.Main.Version, ok)
}

// resolve maps a build-info module version to the version we display, returning
// the fallback for missing, development, or pseudo-version builds.
func resolve(version string, ok bool) string {
	if !ok || version == "" || version == "(devel)" || strings.HasPrefix(version, "v0.0.0-") {
		return fallback
	}
	return version
}
