# quash-board Phase 1: Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the three pure, fully unit-tested foundation packages — `config` (YAML parsing + validation + defaults), `runner` (run a shell command with timeout + env injection), and `layout` (compute widget rectangles from the grid spec) — with no UI yet.

**Architecture:** Three independent packages with no dependencies on Bubble Tea. `config` parses YAML into typed structs via an intermediate raw struct so all validation lives in one place. `runner` wraps `exec.CommandContext` and encodes failures into a `Result` value (never returns an error for script failures). `layout` is a pure function from `(config, width, height)` to a grid of rectangles.

**Tech Stack:** Go 1.26, `gopkg.in/yaml.v3`. Module path: `github.com/locle97/quash-board`.

---

### Task 0: Project initialization

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Initialize the module and add the YAML dependency**

```bash
cd /home/locle97/coding/github/quash-board
go mod init github.com/locle97/quash-board
go get gopkg.in/yaml.v3@latest
```

- [ ] **Step 2: Verify it builds**

Run: `go build ./...`
Expected: succeeds with no output (no packages yet, exits 0).

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: initialize go module with yaml dependency

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 1: Config types and size parsing

The `Size` type represents either a fixed line/column count or an `fr` weight. `parseSize` turns strings like `"1fr"` or `"10"` into a `Size`.

**Files:**
- Create: `config/config.go`
- Test: `config/size_test.go`

- [ ] **Step 1: Write the failing test**

```go
package config

import "testing"

func TestParseSize(t *testing.T) {
	tests := []struct {
		in      string
		want    Size
		wantErr bool
	}{
		{"1fr", Size{Weight: 1}, false},
		{"3fr", Size{Weight: 3}, false},
		{"10", Size{Fixed: 10}, false},
		{"0fr", Size{}, true},
		{"-2", Size{}, true},
		{"abc", Size{}, true},
		{"0", Size{}, true},
	}
	for _, tt := range tests {
		got, err := parseSize(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseSize(%q): expected error, got %+v", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSize(%q): unexpected error %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("parseSize(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./config/`
Expected: FAIL — `undefined: Size`, `undefined: parseSize`.

- [ ] **Step 3: Write the types and parseSize**

```go
package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Size is either a fixed line/column count (Fixed > 0) or an fr weight
// (Weight > 0). Exactly one is non-zero.
type Size struct {
	Fixed  int
	Weight int
}

// Widget is one box on the dashboard.
type Widget struct {
	Name     string
	Title    string
	Script   string
	Interval time.Duration
	Timeout  time.Duration
	Width    Size
}

// Row is a full-width horizontal band holding widgets laid out left-to-right.
type Row struct {
	Height  Size
	Widgets []Widget
}

// Config is the validated dashboard configuration.
type Config struct {
	DefaultInterval time.Duration
	DefaultTimeout  time.Duration
	Rows            []Row
}

func parseSize(s string) (Size, error) {
	if strings.HasSuffix(s, "fr") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "fr"))
		if err != nil || n <= 0 {
			return Size{}, fmt.Errorf("invalid weight %q", s)
		}
		return Size{Weight: n}, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return Size{}, fmt.Errorf("invalid size %q", s)
	}
	return Size{Fixed: n}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./config/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add config/config.go config/size_test.go
git commit -m "feat(config): add typed config structs and size parsing

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: Config loading, validation, and defaults

`Load` reads YAML, rejects unknown fields, applies default interval/timeout, validates required fields, and returns a typed `*Config`.

**Files:**
- Modify: `config/config.go`
- Test: `config/config_test.go`

- [ ] **Step 1: Write the failing test**

```go
package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadAppliesDefaults(t *testing.T) {
	in := `
rows:
  - widgets:
      - name: cpu
        script: "echo hi"
`
	cfg, err := Load(strings.NewReader(in))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultInterval != 5*time.Second {
		t.Errorf("DefaultInterval = %v, want 5s", cfg.DefaultInterval)
	}
	if cfg.DefaultTimeout != 10*time.Second {
		t.Errorf("DefaultTimeout = %v, want 10s", cfg.DefaultTimeout)
	}
	w := cfg.Rows[0].Widgets[0]
	if w.Interval != 5*time.Second || w.Timeout != 10*time.Second {
		t.Errorf("widget defaults not applied: %+v", w)
	}
	if w.Title != "cpu" {
		t.Errorf("Title = %q, want %q (defaults to name)", w.Title, "cpu")
	}
	if w.Width != (Size{Weight: 1}) {
		t.Errorf("Width = %+v, want default weight 1", w.Width)
	}
	if cfg.Rows[0].Height != (Size{Weight: 1}) {
		t.Errorf("Height = %+v, want default weight 1", cfg.Rows[0].Height)
	}
}

func TestLoadOverrides(t *testing.T) {
	in := `
default_interval: 3s
default_timeout: 7s
rows:
  - height: 2fr
    widgets:
      - name: cpu
        title: Processor
        script: "top -bn1"
        interval: 1s
        timeout: 2s
        width: 3fr
`
	cfg, err := Load(strings.NewReader(in))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w := cfg.Rows[0].Widgets[0]
	if w.Interval != time.Second || w.Timeout != 2*time.Second {
		t.Errorf("overrides not applied: %+v", w)
	}
	if w.Title != "Processor" {
		t.Errorf("Title = %q, want Processor", w.Title)
	}
	if w.Width != (Size{Weight: 3}) || cfg.Rows[0].Height != (Size{Weight: 2}) {
		t.Errorf("sizes not parsed: w=%+v h=%+v", w.Width, cfg.Rows[0].Height)
	}
}

func TestLoadErrors(t *testing.T) {
	cases := map[string]string{
		"no rows":          `default_interval: 5s`,
		"widget no name":   "rows:\n  - widgets:\n      - script: \"x\"\n",
		"widget no script": "rows:\n  - widgets:\n      - name: cpu\n",
		"bad duration":     "rows:\n  - widgets:\n      - {name: c, script: x, interval: fast}\n",
		"unknown field":    "rows:\n  - widgets:\n      - {name: c, script: x, color: red}\n",
		"bad height":       "rows:\n  - {height: tall, widgets: [{name: c, script: x}]}\n",
	}
	for name, in := range cases {
		if _, err := Load(strings.NewReader(in)); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./config/ -run TestLoad`
Expected: FAIL — `undefined: Load`.

- [ ] **Step 3: Add raw structs, Load, LoadFile, and convert**

Append to `config/config.go` (add `"io"`, `"os"`, `"gopkg.in/yaml.v3"` to the imports):

```go
const (
	defaultIntervalFallback = 5 * time.Second
	defaultTimeoutFallback  = 10 * time.Second
)

type rawConfig struct {
	DefaultInterval string      `yaml:"default_interval"`
	DefaultTimeout  string      `yaml:"default_timeout"`
	Rows            []rawRow    `yaml:"rows"`
}

type rawRow struct {
	Height  string      `yaml:"height"`
	Widgets []rawWidget `yaml:"widgets"`
}

type rawWidget struct {
	Name     string `yaml:"name"`
	Title    string `yaml:"title"`
	Script   string `yaml:"script"`
	Interval string `yaml:"interval"`
	Timeout  string `yaml:"timeout"`
	Width    string `yaml:"width"`
}

// Load parses and validates a config from r, applying defaults.
func Load(r io.Reader) (*Config, error) {
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	var raw rawConfig
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return convert(raw)
}

// LoadFile loads a config from a file path.
func LoadFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Load(f)
}

func convert(raw rawConfig) (*Config, error) {
	cfg := &Config{
		DefaultInterval: defaultIntervalFallback,
		DefaultTimeout:  defaultTimeoutFallback,
	}
	if raw.DefaultInterval != "" {
		d, err := time.ParseDuration(raw.DefaultInterval)
		if err != nil {
			return nil, fmt.Errorf("default_interval: %w", err)
		}
		cfg.DefaultInterval = d
	}
	if raw.DefaultTimeout != "" {
		d, err := time.ParseDuration(raw.DefaultTimeout)
		if err != nil {
			return nil, fmt.Errorf("default_timeout: %w", err)
		}
		cfg.DefaultTimeout = d
	}
	if len(raw.Rows) == 0 {
		return nil, fmt.Errorf("config must define at least one row")
	}
	hasWidget := false
	for ri, rr := range raw.Rows {
		row := Row{Height: Size{Weight: 1}}
		if rr.Height != "" {
			s, err := parseSize(rr.Height)
			if err != nil {
				return nil, fmt.Errorf("rows[%d].height: %w", ri, err)
			}
			row.Height = s
		}
		for wi, rw := range rr.Widgets {
			if rw.Name == "" {
				return nil, fmt.Errorf("rows[%d].widgets[%d]: name is required", ri, wi)
			}
			if rw.Script == "" {
				return nil, fmt.Errorf("widget %q: script is required", rw.Name)
			}
			w := Widget{
				Name:     rw.Name,
				Title:    rw.Title,
				Script:   rw.Script,
				Interval: cfg.DefaultInterval,
				Timeout:  cfg.DefaultTimeout,
				Width:    Size{Weight: 1},
			}
			if w.Title == "" {
				w.Title = rw.Name
			}
			if rw.Interval != "" {
				d, err := time.ParseDuration(rw.Interval)
				if err != nil {
					return nil, fmt.Errorf("widget %q interval: %w", rw.Name, err)
				}
				w.Interval = d
			}
			if rw.Timeout != "" {
				d, err := time.ParseDuration(rw.Timeout)
				if err != nil {
					return nil, fmt.Errorf("widget %q timeout: %w", rw.Name, err)
				}
				w.Timeout = d
			}
			if rw.Width != "" {
				s, err := parseSize(rw.Width)
				if err != nil {
					return nil, fmt.Errorf("widget %q width: %w", rw.Name, err)
				}
				w.Width = s
			}
			row.Widgets = append(row.Widgets, w)
			hasWidget = true
		}
		cfg.Rows = append(cfg.Rows, row)
	}
	if !hasWidget {
		return nil, fmt.Errorf("config must define at least one widget")
	}
	return cfg, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./config/`
Expected: PASS (all tests).

- [ ] **Step 5: Commit**

```bash
git add config/config.go config/config_test.go
git commit -m "feat(config): add YAML loading, validation, and defaults

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: Script runner

`runner.Run` executes a command via `bash -c` with a timeout and injected env vars, returning a `Result`. Script failures (non-zero exit, timeout) are encoded in the `Result`, never returned as a Go error.

**Files:**
- Create: `runner/runner.go`
- Test: `runner/runner_test.go`

- [ ] **Step 1: Write the failing test**

```go
package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunSuccess(t *testing.T) {
	res := Run(context.Background(), "echo hello", time.Second, Env{})
	if !res.OK() {
		t.Fatalf("expected OK, got %+v", res)
	}
	if strings.TrimSpace(res.Stdout) != "hello" {
		t.Errorf("Stdout = %q, want hello", res.Stdout)
	}
}

func TestRunNonZeroExit(t *testing.T) {
	res := Run(context.Background(), "echo oops >&2; exit 3", time.Second, Env{})
	if res.OK() {
		t.Fatal("expected failure")
	}
	if res.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want 3", res.ExitCode)
	}
	if !strings.Contains(res.Stderr, "oops") {
		t.Errorf("Stderr = %q, want to contain oops", res.Stderr)
	}
}

func TestRunTimeout(t *testing.T) {
	res := Run(context.Background(), "sleep 5", 100*time.Millisecond, Env{})
	if !res.TimedOut {
		t.Errorf("expected TimedOut, got %+v", res)
	}
	if res.OK() {
		t.Error("timed-out result must not be OK")
	}
}

func TestRunInjectsEnv(t *testing.T) {
	res := Run(context.Background(), `echo "$WIDGET_W x $WIDGET_H"`, time.Second,
		Env{WidgetW: 30, WidgetH: 10})
	if strings.TrimSpace(res.Stdout) != "30 x 10" {
		t.Errorf("Stdout = %q, want '30 x 10'", res.Stdout)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./runner/`
Expected: FAIL — `undefined: Run`, `undefined: Env`.

- [ ] **Step 3: Write the runner**

```go
package runner

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// Result captures the outcome of one script run.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	TimedOut bool
	Duration time.Duration
}

// OK reports whether the run succeeded (exit 0, not timed out).
func (r Result) OK() bool { return !r.TimedOut && r.ExitCode == 0 }

// Env holds the values injected into a script's environment.
type Env struct {
	WidgetW, WidgetH int
	Cols, Lines      int
}

// Run executes script via `bash -c` with the given timeout and env vars.
// Failures are encoded in the returned Result, not as a Go error.
func Run(ctx context.Context, script string, timeout time.Duration, env Env) Result {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", script)
	cmd.Env = append(os.Environ(),
		"WIDGET_W="+strconv.Itoa(env.WidgetW),
		"WIDGET_H="+strconv.Itoa(env.WidgetH),
		"COLUMNS="+strconv.Itoa(env.Cols),
		"LINES="+strconv.Itoa(env.Lines),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	res := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(start),
	}
	if ctx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		res.ExitCode = -1
		return res
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.ExitCode = -1
			if res.Stderr != "" {
				res.Stderr += "\n"
			}
			res.Stderr += err.Error()
		}
	}
	return res
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./runner/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add runner/runner.go runner/runner_test.go
git commit -m "feat(runner): run shell commands with timeout and env injection

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 4: Layout engine

`layout.Compute` turns a config plus terminal dimensions into a grid of rectangles, one per widget. `distribute` is the shared weight/fixed-size splitter used for both rows (vertical) and widgets within a row (horizontal).

**Files:**
- Create: `layout/layout.go`
- Test: `layout/layout_test.go`

- [ ] **Step 1: Write the failing test**

```go
package layout

import (
	"testing"

	"github.com/locle97/quash-board/config"
)

func sz(weight, fixed int) config.Size { return config.Size{Weight: weight, Fixed: fixed} }

func TestDistributeEqualWeights(t *testing.T) {
	got := distribute([]config.Size{sz(1, 0), sz(1, 0)}, 20)
	want := []int{10, 10}
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDistributeRemainderToEarliest(t *testing.T) {
	got := distribute([]config.Size{sz(1, 0), sz(1, 0), sz(1, 0)}, 20)
	want := []int{7, 7, 6} // remainder 2 goes to the first two
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDistributeWeighted(t *testing.T) {
	got := distribute([]config.Size{sz(1, 0), sz(2, 0)}, 30)
	want := []int{10, 20}
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDistributeFixedAndWeighted(t *testing.T) {
	// one fixed 5-line row, rest split between two weighted rows
	got := distribute([]config.Size{sz(0, 5), sz(1, 0), sz(1, 0)}, 25)
	want := []int{5, 10, 10}
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestComputeTilesExactly(t *testing.T) {
	cfg := &config.Config{
		Rows: []config.Row{
			{Height: sz(1, 0), Widgets: []config.Widget{
				{Name: "a", Width: sz(1, 0)}, {Name: "b", Width: sz(1, 0)},
			}},
			{Height: sz(1, 0), Widgets: []config.Widget{
				{Name: "c", Width: sz(1, 0)},
			}},
		},
	}
	rects := Compute(cfg, 40, 20)

	// Row 0: two 20-wide boxes, height 10, at y=0
	if rects[0][0] != (Rect{X: 0, Y: 0, W: 20, H: 10}) {
		t.Errorf("rects[0][0] = %+v", rects[0][0])
	}
	if rects[0][1] != (Rect{X: 20, Y: 0, W: 20, H: 10}) {
		t.Errorf("rects[0][1] = %+v", rects[0][1])
	}
	// Row 1: one full-width box, height 10, at y=10
	if rects[1][0] != (Rect{X: 0, Y: 10, W: 40, H: 10}) {
		t.Errorf("rects[1][0] = %+v", rects[1][0])
	}
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./layout/`
Expected: FAIL — `undefined: distribute`, `undefined: Compute`, `undefined: Rect`.

- [ ] **Step 3: Write the layout engine**

```go
package layout

import "github.com/locle97/quash-board/config"

// Rect is a widget's box position and size, in terminal cells.
type Rect struct {
	X, Y, W, H int
}

// Compute returns rectangles aligned with cfg.Rows/Widgets: rects[r][c] is the
// box for the widget at row r, column c. The grid fills termW x termH exactly.
func Compute(cfg *config.Config, termW, termH int) [][]Rect {
	rowSizes := make([]config.Size, len(cfg.Rows))
	for i, r := range cfg.Rows {
		rowSizes[i] = r.Height
	}
	rowHeights := distribute(rowSizes, termH)

	out := make([][]Rect, len(cfg.Rows))
	y := 0
	for ri, row := range cfg.Rows {
		h := rowHeights[ri]
		colSizes := make([]config.Size, len(row.Widgets))
		for i, w := range row.Widgets {
			colSizes[i] = w.Width
		}
		colWidths := distribute(colSizes, termW)

		rects := make([]Rect, len(row.Widgets))
		x := 0
		for ci := range row.Widgets {
			w := colWidths[ci]
			rects[ci] = Rect{X: x, Y: y, W: w, H: h}
			x += w
		}
		out[ri] = rects
		y += h
	}
	return out
}

// distribute splits total across sizes. Fixed sizes take their value; the
// remaining space is split among weighted sizes proportionally, with any
// rounding remainder handed to the earliest weighted entries.
func distribute(sizes []config.Size, total int) []int {
	res := make([]int, len(sizes))
	fixedTotal, weightTotal := 0, 0
	var weighted []int
	for i, s := range sizes {
		if s.Fixed > 0 {
			res[i] = s.Fixed
			fixedTotal += s.Fixed
		} else {
			weighted = append(weighted, i)
			weightTotal += s.Weight
		}
	}
	remaining := total - fixedTotal
	if remaining < 0 {
		remaining = 0
	}
	if weightTotal == 0 {
		return res
	}
	assigned := 0
	for _, i := range weighted {
		share := remaining * sizes[i].Weight / weightTotal
		res[i] = share
		assigned += share
	}
	for k := 0; k < remaining-assigned; k++ {
		res[weighted[k]]++
	}
	return res
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./layout/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add layout/layout.go layout/layout_test.go
git commit -m "feat(layout): compute widget rectangles from grid spec

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 5: Phase 1 verification

- [ ] **Step 1: Run the full test suite with vet**

Run: `go vet ./... && go test ./...`
Expected: all packages PASS, vet clean.

- [ ] **Step 2: Confirm tidy modules**

Run: `go mod tidy && git diff --exit-code go.mod go.sum`
Expected: no changes (already tidy). If `go.sum` changes, commit it.

**Phase 1 deliverable:** `config`, `runner`, and `layout` packages, each unit-tested. The next phase builds rendering and print mode on top of these.
