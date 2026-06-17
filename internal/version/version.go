// Package version exposes the build version of quash-board.
package version

import "runtime/debug"

// fallback is reported for builds without an embedded module version, such as
// local `go build` or `go run` (where ReadBuildInfo reports "(devel)" or "").
const fallback = "dev"

// Version returns the module version this binary was built from. Binaries
// installed via `go install <module>@vX.Y.Z` report the tag (e.g. "v0.1.0");
// local builds report "dev".
func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return fallback
	}
	return info.Main.Version
}
