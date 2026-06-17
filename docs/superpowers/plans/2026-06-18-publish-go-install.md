# Publish quash-board for `go install` — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `go install github.com/locle97/quash-board@latest` install a working `quash-board` binary that reports its installed version.

**Architecture:** Migrate the bare module path `quash-board` to the resolvable `github.com/locle97/quash-board`, rewriting all internal imports. Replace the hardcoded version const with a function that reads the module version from `runtime/debug.ReadBuildInfo()`, and surface it via a `--version` flag. Document the install path, then tag `v0.1.0`.

**Tech Stack:** Go 1.25, standard library (`flag`, `runtime/debug`), git.

**Prerequisite (already done):** The GitHub repo has been renamed `squash-board` → `quash-board` and the local `origin` remote points to `https://github.com/locle97/quash-board.git`.

---

### Task 1: Migrate module path and internal imports

**Files:**
- Modify: `go.mod:1`
- Modify (imports): `main.go`, `internal/layout/layout.go`, `internal/widget/widget.go`, `internal/printer/printer.go`, `internal/tui/tui.go`, `internal/tui/tui_test.go`, `internal/printer/printer_test.go`, `internal/widget/widget_test.go`, `internal/layout/compute_test.go`

- [ ] **Step 1: Confirm the current import sites**

Run: `grep -rl 'quash-board/internal' --include='*.go' .`
Expected: the 9 files listed above (order may vary).

- [ ] **Step 2: Change the module path in go.mod**

Edit `go.mod` line 1:

```
module github.com/locle97/quash-board
```

(Leave the `go 1.25.0` line and all `require` blocks unchanged.)

- [ ] **Step 3: Rewrite all internal imports**

Run:

```bash
grep -rl 'quash-board/internal' --include='*.go' . \
  | xargs sed -i 's#"quash-board/internal#"github.com/locle97/quash-board/internal#g'
```

- [ ] **Step 4: Verify no bare-path imports remain**

Run: `grep -rn '"quash-board/internal' --include='*.go' .`
Expected: no output (exit status 1).

- [ ] **Step 5: Verify the build and tests pass**

Run: `go build ./... && go test ./...`
Expected: build succeeds; all packages `ok`.

- [ ] **Step 6: Commit**

```bash
git add go.mod main.go internal/
git commit -m "refactor: migrate module path to github.com/locle97/quash-board"
```

---

### Task 2: Read version from build info

Replace the hardcoded const with a function that returns the embedded module version, falling back to `"dev"` for local (non-installed) builds.

**Files:**
- Modify: `internal/version/version.go`
- Test: `internal/version/version_test.go`

- [ ] **Step 1: Write the failing tests**

Replace the entire contents of `internal/version/version_test.go` with:

```go
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/version/`
Expected: FAIL — `Version` is a const, not a function (`Version()` does not compile; `fallback` undefined).

- [ ] **Step 3: Rewrite the version package**

Replace the entire contents of `internal/version/version.go` with:

```go
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
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/version/`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add internal/version/
git commit -m "feat(version): read version from build info with dev fallback"
```

---

### Task 3: Add a `--version` flag

Wire the version package into `main.go` so installed users can check their version.

**Files:**
- Modify: `main.go:5-17` (imports), `main.go:20-24` (flags), `main.go:24-25` (after `flag.Parse()`)

- [ ] **Step 1: Add the version import**

In the import block of `main.go`, add (keeping imports grouped/sorted with the other internal imports):

```go
	"github.com/locle97/quash-board/internal/version"
```

So the internal import group reads:

```go
	"github.com/locle97/quash-board/internal/config"
	"github.com/locle97/quash-board/internal/printer"
	"github.com/locle97/quash-board/internal/tui"
	"github.com/locle97/quash-board/internal/version"
```

- [ ] **Step 2: Add the `--version` flag declaration**

In `main()`, alongside the other `flag.*` declarations (after the `noColor` line), add:

```go
	showVersion := flag.Bool("version", false, "print version and exit")
```

- [ ] **Step 3: Handle the flag after parsing**

Immediately after `flag.Parse()`, before the `resolveConfigPath` call, add:

```go
	if *showVersion {
		fmt.Println(version.Version())
		return
	}
```

- [ ] **Step 4: Verify build, vet, and tests**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: build succeeds; vet clean; all packages `ok`.

- [ ] **Step 5: Verify the flag works at runtime**

Run: `go run . --version`
Expected: prints `dev` (local builds report the fallback).

- [ ] **Step 6: Commit**

```bash
git add main.go
git commit -m "feat: add --version flag"
```

---

### Task 4: Document the install path in README

**Files:**
- Modify: `README.md:6-10` (the `## Install` section)

- [ ] **Step 1: Replace the Install section**

Replace lines 6–10 (the current `## Install` heading and its build-from-source code block) with:

```markdown
## Install

With a Go toolchain (1.25+):

```bash
go install github.com/locle97/quash-board@latest
```

This installs the `quash-board` binary into `$(go env GOPATH)/bin` (or `$GOBIN`
if set), which must be on your `$PATH`. Pin a specific version with, e.g.,
`go install github.com/locle97/quash-board@v0.1.0`.

### Build from source

```bash
git clone https://github.com/locle97/quash-board.git
cd quash-board
go build -o quash-board .
```
```

- [ ] **Step 2: Verify the rendered section reads correctly**

Run: `sed -n '6,30p' README.md`
Expected: an Install section with the `go install` command followed by a "Build from source" subsection.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: document go install path"
```

---

### Task 5: Push and tag the release

This makes the module resolvable on the Go proxy and `@latest` point at `v0.1.0`.

**Files:** none (git operations only).

- [ ] **Step 1: Push the committed work to the renamed remote**

Run: `git push origin master`
Expected: push succeeds to `github.com/locle97/quash-board`.

- [ ] **Step 2: Create the annotated tag**

Run: `git tag -a v0.1.0 -m "quash-board v0.1.0"`
Expected: no output.

- [ ] **Step 3: Push the tag**

Run: `git push origin v0.1.0`
Expected: `* [new tag] v0.1.0 -> v0.1.0`.

- [ ] **Step 4: Verify go install resolves the tag (clean module cache)**

Run:

```bash
GOBIN=$(mktemp -d) go install github.com/locle97/quash-board@v0.1.0 && \
  GOBIN_DIR=$(ls -d /tmp/tmp.*/quash-board 2>/dev/null | head -1)
```

Simpler equivalent, then check the version:

```bash
TMPBIN=$(mktemp -d); GOBIN=$TMPBIN go install github.com/locle97/quash-board@v0.1.0; "$TMPBIN/quash-board" --version
```

Expected: `--version` prints `v0.1.0`.

> Note: the proxy may take up to a minute to index a brand-new tag. If the install fails to resolve, retry after a short wait or use `GOPROXY=direct`.

---

## Self-Review Notes

- **Spec coverage:** Module path fix → Task 1. Version from build info + `--version` flag → Tasks 2–3. README install instructions → Task 4. Release tagging `v0.1.0` → Task 5. All spec sections covered.
- **Type consistency:** `version.Version()` (function) is defined in Task 2 and called in Task 3 and the runtime check in Task 5. `fallback` const used in Task 2 test and implementation.
- **Prerequisite:** Repo rename + remote URL update are done (noted in header), so Task 5's push/tag will resolve to the correct path.
