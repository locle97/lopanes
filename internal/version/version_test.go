package version

import "testing"

func TestVersionNotEmpty(t *testing.T) {
	if Version() == "" {
		t.Fatal("Version() must not be empty")
	}
}

func TestVersionFallsBackForLocalBuild(t *testing.T) {
	// Under `go test`, the build carries no embedded module version, so
	// Version() must return the fallback rather than an empty or "(devel)" string.
	if got := Version(); got != fallback {
		t.Fatalf("Version() = %q, want fallback %q for a non-installed build", got, fallback)
	}
}
