# Publish quash-board for `go install` — Design

**Date:** 2026-06-18
**Status:** Approved

## Goal

Let users install quash-board with `go install github.com/locle97/quash-board@latest`
instead of cloning and building from source.

## Scope

Minimal, `go-install`-first. No CI release automation, no prebuilt binaries, no
Homebrew tap. Those can come later.

## Background / Current State

- `go.mod` declares a bare module path: `module quash-board`. This is not a
  resolvable import path, so `go install` cannot fetch it.
- The GitHub repo is `github.com/locle97/squash-board`, but the module, binary,
  and brand are all `quash-board` — a name mismatch.
- `internal/version/version.go` defines `const Version = "0.1.0"`, but it is not
  referenced anywhere in `main.go` (dead code) and there is no `--version` flag.
- A hardcoded version const cannot reflect the tag a user installed via
  `go install`.

## Decisions

- **Name:** Rename the GitHub repo `squash-board` → `quash-board` so the module,
  binary, repo, and brand all match. The installed binary name is the last
  segment of the module path, so this yields a `quash-board` command.
- **Version source:** Read the version from build info at runtime rather than a
  hardcoded const.
- **First tag:** `v0.1.0`.

## Design

### 1. Module path fix (core change)

- User renames the GitHub repo `squash-board` → `quash-board` (manual; requires
  GitHub account access — Claude cannot do this).
- Update the local git remote URL to the renamed repo.
- Change `go.mod`: `module quash-board` → `module github.com/locle97/quash-board`.
- Rewrite all internal imports from `quash-board/internal/...` to
  `github.com/locle97/quash-board/internal/...` across `main.go` and every
  package and test file.
- Verify with `go build ./...` and `go test ./...`.

After this, `go install github.com/locle97/quash-board@latest` produces a
`quash-board` binary in `$GOBIN` (or `$GOPATH/bin`).

### 2. Version from build info

- Replace `const Version = "0.1.0"` with a function (e.g. `version.Version()`)
  that reads `runtime/debug.ReadBuildInfo()` and returns `info.Main.Version`.
  - For `go install module@vX.Y.Z` builds, this returns the real tag
    (e.g. `v0.1.0`).
  - For local `go build` / `go run`, `Main.Version` is `(devel)` or empty;
    fall back to a default string `"dev"`.
- Add a `--version` flag to `main.go` that prints the version and exits. This
  also makes the version package live code instead of dead code.

### 3. Release tagging

- Create and push an annotated semver tag `v0.1.0`. The Go module proxy requires
  a `vMAJOR.MINOR.PATCH` tag for `@latest` to resolve to a stable release.
- Tag is pushed only after the module path fix is committed and on the default
  branch.

### 4. README install instructions

- Add an **Install** section:
  - `go install github.com/locle97/quash-board@latest`
  - Note that the binary lands in `$GOBIN` / `$(go env GOPATH)/bin`, which must
    be on `$PATH`.
  - Show pinned-version form: `go install github.com/locle97/quash-board@v0.1.0`.
- Keep build-from-source as a fallback section.

## Out of Scope

- GitHub Actions release workflow.
- Prebuilt cross-platform binaries / GoReleaser.
- Homebrew tap or other package managers.

## Success Criteria

- `go install github.com/locle97/quash-board@latest` installs a working
  `quash-board` binary on a clean machine.
- `quash-board --version` prints `v0.1.0`.
- `go build ./...` and `go test ./...` pass after the import rewrite.
- README documents the install path.

## Risks / Notes

- The repo rename is a manual prerequisite. Until it is done and the remote URL
  updated, the tag push and module resolution will not work. The import rewrite
  and version changes can be done before the rename, but the `v0.1.0` tag should
  be pushed only after the rename so the proxy resolves the correct path.
