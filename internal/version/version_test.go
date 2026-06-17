package version

import "testing"

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		version string
		ok      bool
		want    string
	}{
		{"no build info", "", false, fallback},
		{"empty version", "", true, fallback},
		{"devel marker", "(devel)", true, fallback},
		{"local pseudo-version", "v0.0.0-20260617174314-212de90678ae", true, fallback},
		{"dirty pseudo-version", "v0.0.0-20260617174314-212de90678ae+dirty", true, fallback},
		{"released tag", "v0.1.0", true, "v0.1.0"},
		{"released tag dirty", "v0.1.0+dirty", true, "v0.1.0+dirty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolve(tt.version, tt.ok); got != tt.want {
				t.Errorf("resolve(%q, %v) = %q, want %q", tt.version, tt.ok, got, tt.want)
			}
		})
	}
}

func TestVersionNotEmpty(t *testing.T) {
	if Version() == "" {
		t.Fatal("Version() must not be empty")
	}
}
