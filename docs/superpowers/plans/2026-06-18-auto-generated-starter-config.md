# Auto-generated Starter Config Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When lopanes finds no config, generate a working starter config at `~/.config/lopanes/config.yaml`, tell the user, and run it — so first run shows a dashboard instead of an error.

**Architecture:** Embed a portable starter YAML into the binary via `go:embed` as the single source of truth. A new `config.WriteDefault` writes it to disk. `main.go` replaces `resolveConfigPath` with a `loadConfig` helper that, on no-config, writes the starter file and loads it — falling back to parsing the embedded bytes in memory if the write fails.

**Tech Stack:** Go 1.25, standard library (`embed`, `os`, `path/filepath`), existing `internal/config` package.

## Global Constraints

- Go version floor: `go 1.25.0` (from `go.mod`).
- Binary/module name: `lopanes` (`github.com/locle97/lopanes`).
- `config.Parse` and `config.Load` return `config.Config` by value (not a pointer).
- Starter config must be portable: inline shell only, no external script files; works on Linux and macOS.
- stderr messages are prefixed `lopanes:` (matches existing `fail`).
- Generation only happens for the no-flag, no-config case. An explicit missing `--config PATH` still errors. Both interactive and `--print` modes generate.

---

### Task 1: Embedded starter template and `WriteDefault`

**Files:**
- Create: `internal/config/default.yaml`
- Create: `internal/config/default.go`
- Test: `internal/config/default_test.go`

**Interfaces:**
- Consumes: `config.Parse([]byte) (Config, error)` (existing).
- Produces:
  - `config.DefaultYAML []byte` — embedded starter config bytes.
  - `config.WriteDefault(path string) error` — creates parent dirs and writes `DefaultYAML` to `path` (mode `0644`); errors if `path` already exists.

- [ ] **Step 1: Create the starter template file**

Create `internal/config/default.yaml`:

```yaml
default_interval: 2s
default_timeout: 5s
rows:
  - height: 1fr
    widgets:
      - {name: clock, title: "Clock", script: "date '+%H:%M:%S'", interval: 1s}
      - {name: uptime, title: "Uptime", script: "uptime"}
  - height: 2fr
    widgets:
      - {name: disk, title: "Disk", script: "df -h / 2>/dev/null | head -n 5", width: 2fr}
      - {name: mem, title: "Memory", script: "free -h 2>/dev/null || vm_stat"}
```

- [ ] **Step 2: Write the failing tests**

Create `internal/config/default_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultYAMLParses(t *testing.T) {
	cfg, err := Parse(DefaultYAML)
	if err != nil {
		t.Fatalf("default config does not parse: %v", err)
	}
	if len(cfg.Rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(cfg.Rows))
	}
	if len(cfg.Rows[0].Widgets) != 2 || len(cfg.Rows[1].Widgets) != 2 {
		t.Fatalf("want 2 widgets per row, got %d and %d",
			len(cfg.Rows[0].Widgets), len(cfg.Rows[1].Widgets))
	}
}

func TestWriteDefaultCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.yaml")
	if err := WriteDefault(path); err != nil {
		t.Fatalf("WriteDefault: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(got) != string(DefaultYAML) {
		t.Fatal("written content does not match DefaultYAML")
	}
}

func TestWriteDefaultRefusesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteDefault(path); err == nil {
		t.Fatal("WriteDefault should error when target exists")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/config/ -run 'Default|WriteDefault' -v`
Expected: FAIL — `undefined: DefaultYAML` / `undefined: WriteDefault` (build error).

- [ ] **Step 4: Write the implementation**

Create `internal/config/default.go`:

```go
package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultYAML is the built-in starter dashboard configuration, used both as
// the file written on first run and as the in-memory fallback when that write
// fails. It is the single source of truth for the default config.
//
//go:embed default.yaml
var DefaultYAML []byte

// WriteDefault writes the starter config to path, creating parent directories
// as needed. It refuses to overwrite an existing file.
func WriteDefault(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("creating config file: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(DefaultYAML); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`
Expected: PASS (all config tests, including the three new ones).

- [ ] **Step 6: Commit**

```bash
git add internal/config/default.yaml internal/config/default.go internal/config/default_test.go
git commit -m "feat(config): embed starter config and add WriteDefault"
```

---

### Task 2: Wire auto-generation into startup

**Files:**
- Modify: `main.go` (replace `resolveConfigPath` usage and definition with `loadConfig`)
- Modify: `main_test.go` (update tests to target `loadConfig`)
- Modify: `README.md` (document the behavior)

**Interfaces:**
- Consumes: `config.DefaultYAML`, `config.WriteDefault`, `config.Load`, `config.Parse` from Task 1.
- Produces: `loadConfig(flagPath string) (config.Config, error)` — resolves and loads the config, generating the starter config when none is found.

- [ ] **Step 1: Update the tests to target `loadConfig`**

Replace the entire contents of `main_test.go` with:

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

const validYAML = `
rows:
  - height: 1fr
    widgets:
      - {name: a, script: "echo hi"}
`

func TestLoadConfigExplicit(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "my.yaml")
	if err := os.WriteFile(p, []byte(validYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(p)
	if err != nil {
		t.Fatalf("explicit path: err %v", err)
	}
	if len(cfg.Rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(cfg.Rows))
	}
}

func TestLoadConfigExplicitMissing(t *testing.T) {
	if _, err := loadConfig("/no/such/file.yaml"); err == nil {
		t.Fatal("missing explicit path should error")
	}
}

func TestLoadConfigSearchesCwd(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("lopanes.yaml", []byte(validYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(""); err != nil {
		t.Fatalf("search err: %v", err)
	}
}

func TestLoadConfigGeneratesWhenNoneFound(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", dir) // no ~/.config/lopanes/config.yaml here yet

	cfg, err := loadConfig("")
	if err != nil {
		t.Fatalf("expected generation to succeed, got err %v", err)
	}
	if len(cfg.Rows) == 0 {
		t.Fatal("generated config should have rows")
	}
	gen := filepath.Join(dir, ".config", "lopanes", "config.yaml")
	if _, err := os.Stat(gen); err != nil {
		t.Fatalf("expected starter config written at %s: %v", gen, err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test . -run 'TestLoadConfig' -v`
Expected: FAIL — `undefined: loadConfig` (build error).

- [ ] **Step 3: Replace `resolveConfigPath` with `loadConfig` in `main.go`**

In `main.go`, change the call site. Replace:

```go
	path, err := resolveConfigPath(*cfgPath)
	if err != nil {
		fail(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		fail(err)
	}
```

with:

```go
	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		fail(err)
	}
```

Then replace the `resolveConfigPath` function (and its doc comment) with:

```go
// loadConfig resolves and loads the configuration. An explicit flagPath must
// exist. Otherwise the default search order is ./lopanes.yaml then
// ~/.config/lopanes/config.yaml; when neither exists, a starter config is
// written to the latter and loaded. If that write fails, the embedded default
// is parsed in memory so the first run never hard-fails.
func loadConfig(flagPath string) (config.Config, error) {
	if flagPath != "" {
		if _, err := os.Stat(flagPath); err != nil {
			return config.Config{}, fmt.Errorf("config %q: %w", flagPath, err)
		}
		return config.Load(flagPath)
	}

	candidates := []string{"./lopanes.yaml"}
	var globalPath string
	if home, err := os.UserHomeDir(); err == nil {
		globalPath = filepath.Join(home, ".config", "lopanes", "config.yaml")
		candidates = append(candidates, globalPath)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return config.Load(c)
		}
	}

	if globalPath == "" {
		fmt.Fprintln(os.Stderr, "lopanes: no config found and no home directory; using built-in default")
		return config.Parse(config.DefaultYAML)
	}
	if err := config.WriteDefault(globalPath); err != nil {
		fmt.Fprintf(os.Stderr, "lopanes: no config found and could not write starter config (%v); using built-in default\n", err)
		return config.Parse(config.DefaultYAML)
	}
	fmt.Fprintf(os.Stderr, "lopanes: no config found; wrote a starter config to %s — edit it to customize.\n", globalPath)
	return config.Load(globalPath)
}
```

Remove the now-unused `strings` import if `go build` reports it (it was only used by the old error message).

- [ ] **Step 4: Run tests and build to verify they pass**

Run: `go test . -v && go build ./...`
Expected: PASS for all `main` tests; build succeeds with no unused-import errors.

- [ ] **Step 5: Update README**

In `README.md`, under the `--config PATH` bullet in the Usage section, append a sentence describing generation. Change:

```
- `--config PATH` — config file. Default search order: `./lopanes.yaml`,
  then `~/.config/lopanes/config.yaml`.
```

to:

```
- `--config PATH` — config file. Default search order: `./lopanes.yaml`,
  then `~/.config/lopanes/config.yaml`. If no config is found, lopanes writes
  a starter config to `~/.config/lopanes/config.yaml` and runs it, so the
  first run shows a working dashboard. Edit that file to customize.
```

- [ ] **Step 6: Manually verify the first-run experience**

Run: `HOME=$(mktemp -d) ./lopanes --print --width 100` after `go build -o lopanes .`
Expected: stderr shows `lopanes: no config found; wrote a starter config to <tmp>/.config/lopanes/config.yaml — edit it to customize.` and stdout renders the four-widget dashboard.

- [ ] **Step 7: Commit**

```bash
git add main.go main_test.go README.md
git commit -m "feat: generate starter config on first run when none found"
```

---

## Self-Review

**Spec coverage:**
- Generate-to-disk on no-config → Task 2 `loadConfig` + Task 1 `WriteDefault`. ✓
- stderr message wording → Task 2 Step 3 (verbatim from spec). ✓
- Both interactive and `--print` modes → `loadConfig` is called before the mode branch in `main.go`; Task 2 Step 6 verifies via `--print`. ✓
- Explicit `--config` missing still errors → Task 2 `loadConfig` first branch + `TestLoadConfigExplicitMissing`. ✓
- Config already present is used → cwd/global stat loop + `TestLoadConfigSearchesCwd`. ✓
- Write-failure in-memory fallback → `loadConfig` `WriteDefault` error branch + no-home branch. (Write-failure path is lightly covered by tests due to portability of simulating an unwritable HOME; the `WriteDefault`-refuses-existing test in Task 1 exercises the error return it depends on.) ✓
- Self-contained inline starter config → Task 1 `default.yaml`, validated by `TestDefaultYAMLParses`. ✓
- Embedded single source of truth via `go:embed` → Task 1 `default.go`. ✓

**Placeholder scan:** No TBD/TODO/"handle edge cases"; every code step shows full code. ✓

**Type consistency:** `loadConfig` returns `config.Config` (matches `Parse`/`Load` value returns and `tui.New`/`printer.Render` signatures). `WriteDefault(path string) error` and `DefaultYAML []byte` used consistently across both tasks. ✓
