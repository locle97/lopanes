# Per-pane Border Color Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let users set a `color` per pane (with a top-level `default_color`, defaulting to white) that tints each pane's border and title.

**Architecture:** `config` validates the color string at load time into a canonical lipgloss value (ANSI index or hex), resolving widget → default_color → white precedence. The `widget` package gains a `Colorize` theme hook and a `Color` field on `View`; `Render` wraps the border lines and side bars (not the body) with it. `tui` and `printer` thread each widget's color through.

**Tech Stack:** Go, Bubble Tea, Lip Gloss (`lipgloss.Color` / `Foreground`), gopkg.in/yaml.v3.

## Global Constraints

- Module path: `github.com/locle97/lopanes`.
- Color applies to **border + title only**; body keeps its own ANSI.
- Color is applied regardless of widget state (pending/ok/error); the inner `⚠` indicator stays the sole error signal.
- Canonical `white` is the ANSI string `"7"`.
- `PlainTheme` must emit no color (used by `--no-color` print mode and tests).
- Follow existing error-message style: `rows[%d].widgets[%d].color: ...`.

---

### Task 1: Config schema, color parsing & validation

**Files:**
- Create: `internal/config/color.go`
- Create: `internal/config/color_test.go`
- Modify: `internal/config/config.go` (add fields, resolve precedence)
- Test: `internal/config/config_test.go` (default-color resolution)

**Interfaces:**
- Produces: `func parseColor(s, def string) (string, error)` — returns canonical lipgloss value; `parseColor("", def)` returns `def`.
- Produces: `config.Config.DefaultColor string`, `config.Widget.Color string` (both canonical values).

- [ ] **Step 1: Write the failing color-parser tests**

Create `internal/config/color_test.go`:
```go
package config

import "testing"

func TestParseColorNames(t *testing.T) {
	cases := map[string]string{
		"black": "0", "red": "1", "green": "2", "yellow": "3",
		"blue": "4", "magenta": "5", "cyan": "6", "white": "7",
		"bright-black": "8", "gray": "8", "grey": "8",
		"bright-red": "9", "bright-white": "15",
	}
	for in, want := range cases {
		got, err := parseColor(in, "7")
		if err != nil {
			t.Errorf("parseColor(%q) error: %v", in, err)
		}
		if got != want {
			t.Errorf("parseColor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseColorHexAnd256(t *testing.T) {
	for _, in := range []string{"#fff", "#ff8800", "#FF8800", "0", "255", "33"} {
		got, err := parseColor(in, "7")
		if err != nil {
			t.Errorf("parseColor(%q) error: %v", in, err)
		}
		if got != in {
			t.Errorf("parseColor(%q) = %q, want passthrough", in, got)
		}
	}
}

func TestParseColorEmptyReturnsDefault(t *testing.T) {
	got, err := parseColor("", "4")
	if err != nil || got != "4" {
		t.Fatalf("parseColor(\"\", \"4\") = %q, %v", got, err)
	}
}

func TestParseColorInvalid(t *testing.T) {
	for _, in := range []string{"reed", "256", "-1", "#gg0000", "#ff", "ff8800"} {
		if _, err := parseColor(in, "7"); err == nil {
			t.Errorf("parseColor(%q) expected error, got nil", in)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -run TestParseColor -v`
Expected: FAIL — `undefined: parseColor`.

- [ ] **Step 3: Implement `parseColor`**

Create `internal/config/color.go`:
```go
package config

import (
	"fmt"
	"strconv"
	"strings"
)

// colorNames maps friendly color names to their ANSI 0–15 index (as a string).
var colorNames = map[string]string{
	"black": "0", "red": "1", "green": "2", "yellow": "3",
	"blue": "4", "magenta": "5", "cyan": "6", "white": "7",
	"bright-black": "8", "gray": "8", "grey": "8",
	"bright-red": "9", "bright-green": "10", "bright-yellow": "11",
	"bright-blue": "12", "bright-magenta": "13", "bright-cyan": "14",
	"bright-white": "15",
}

// parseColor validates a pane color spec and returns a canonical
// lipgloss-acceptable string (an ANSI index 0–255 or a hex literal). It accepts
// a friendly name, a bare 0–255 integer, or a #rgb / #rrggbb hex value. An empty
// spec returns def.
func parseColor(s, def string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return def, nil
	}
	if code, ok := colorNames[strings.ToLower(s)]; ok {
		return code, nil
	}
	if strings.HasPrefix(s, "#") {
		if isHexColor(s) {
			return s, nil
		}
		return "", fmt.Errorf("invalid hex color %q", s)
	}
	if n, err := strconv.Atoi(s); err == nil {
		if n < 0 || n > 255 {
			return "", fmt.Errorf("color index out of range 0-255: %q", s)
		}
		return s, nil
	}
	return "", fmt.Errorf("unknown color %q", s)
}

// isHexColor reports whether s is #rgb or #rrggbb (case-insensitive).
func isHexColor(s string) bool {
	hex := strings.TrimPrefix(s, "#")
	if len(hex) != 3 && len(hex) != 6 {
		return false
	}
	for _, r := range hex {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// defaultColor is the canonical fallback when no color is configured anywhere.
const defaultColor = "7" // white
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -run TestParseColor -v`
Expected: PASS (all four tests).

- [ ] **Step 5: Add schema fields and precedence resolution**

In `internal/config/config.go`, add to the `Config` struct (after `DefaultTimeout`):
```go
	DefaultColor    string
```
Add to the `Widget` struct (after `WidthWeight`):
```go
	Color       string // canonical lipgloss color value
```
Add to `rawConfig` (after `DefaultTimeout`):
```go
	DefaultColor    string   `yaml:"default_color"`
```
Add to `rawWidget` (after `Width`):
```go
	Color    string `yaml:"color"`
```
In `(r rawConfig) toConfig()`, after the `dt, err := parseDurationDefault(...)` block, add:
```go
	dc, err := parseColor(r.DefaultColor, defaultColor)
	if err != nil {
		return Config{}, fmt.Errorf("default_color: %w", err)
	}
```
Change the `cfg := Config{...}` line to include the default color:
```go
	cfg := Config{DefaultInterval: di, DefaultTimeout: dt, DefaultColor: dc}
```
Inside the widget loop, after the `width, err := parseWeight(...)` block, add:
```go
			color, err := parseColor(rw.Color, dc)
			if err != nil {
				return Config{}, fmt.Errorf("rows[%d].widgets[%d].color: %w", ri, wi, err)
			}
```
Add `Color: color,` to the `Widget{...}` literal appended to `row.Widgets` (after `WidthWeight: width,`).

- [ ] **Step 6: Write the config-resolution test**

Add to `internal/config/config_test.go`:
```go
func TestParseColorPrecedence(t *testing.T) {
	src := `
default_color: gray
rows:
  - widgets:
      - {name: cpu, color: cyan, script: "echo hi"}
      - {name: mem, script: "echo hi"}
`
	cfg, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultColor != "8" {
		t.Errorf("DefaultColor = %q, want 8", cfg.DefaultColor)
	}
	if got := cfg.Rows[0].Widgets[0].Color; got != "6" {
		t.Errorf("cpu color = %q, want 6 (cyan override)", got)
	}
	if got := cfg.Rows[0].Widgets[1].Color; got != "8" {
		t.Errorf("mem color = %q, want 8 (inherits default)", got)
	}
}

func TestParseColorDefaultsWhite(t *testing.T) {
	src := `
rows:
  - widgets:
      - {name: cpu, script: "echo hi"}
`
	cfg, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultColor != "7" || cfg.Rows[0].Widgets[0].Color != "7" {
		t.Errorf("want white (7), got default=%q widget=%q",
			cfg.DefaultColor, cfg.Rows[0].Widgets[0].Color)
	}
}
```

- [ ] **Step 7: Run the full config package tests**

Run: `go test ./internal/config/ -v`
Expected: PASS, including the existing tests (unchanged behavior) and the new ones.

- [ ] **Step 8: Commit**

```bash
git add internal/config/color.go internal/config/color_test.go internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): parse and validate per-pane color"
```

---

### Task 2: Render colored borders and thread color through tui/printer

**Files:**
- Modify: `internal/widget/widget.go` (`View.Color`, `Theme.Colorize`, `FromResult` signature, `Render`)
- Modify: `internal/widget/widget_test.go` (update callers; add color assertions)
- Modify: `internal/tui/tui.go` (set initial `Color`; pass `w.Color` to `FromResult`)
- Modify: `internal/printer/printer.go` (pass `w.Color` to `FromResult`)

**Interfaces:**
- Consumes: `config.Widget.Color` (canonical value from Task 1).
- Produces: `widget.View{ ..., Color string }`; `widget.Theme{ Dim, Colorize func(s, color string) string }`; `func FromResult(title, color, lastGood string, res runner.Result) (View, string)`.

- [ ] **Step 1: Write the failing render tests**

Add to `internal/widget/widget_test.go`:
```go
func TestRenderColorWrapsBorderNotBody(t *testing.T) {
	v := View{Title: "cpu", State: StateOK, Body: "42%", Color: "6"}
	got := Render(v, layout.Rect{W: 10, H: 4}, DefaultTheme())
	lines := strings.Split(got, "\n")
	const esc = "\x1b"
	// Top and bottom border lines carry color.
	if !strings.Contains(lines[0], esc) || !strings.Contains(lines[3], esc) {
		t.Errorf("expected ANSI on border lines, got:\n%q", got)
	}
	// The body text "42%" itself is not wrapped in color (only the side bars are).
	if !strings.Contains(lines[1], "42%") {
		t.Errorf("body text missing: %q", lines[1])
	}
	if strings.Contains(lines[1], esc+"[0m42%") {
		t.Errorf("body text should not be colorized: %q", lines[1])
	}
}

func TestRenderPlainThemeNoColor(t *testing.T) {
	v := View{Title: "cpu", State: StateOK, Body: "42%", Color: "6"}
	got := Render(v, layout.Rect{W: 10, H: 4}, PlainTheme())
	if strings.Contains(got, "\x1b") {
		t.Errorf("PlainTheme must emit no ANSI, got:\n%q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/widget/ -run TestRenderColor -v`
Expected: FAIL — `View` has no field `Color` (compile error).

- [ ] **Step 3: Add `Color` to `View` and `Colorize` to `Theme`**

In `internal/widget/widget.go`, add to the `View` struct (after `ErrTail`):
```go
	Color    string // canonical lipgloss color for the border/title
```
Add to the `Theme` struct (after `Dim`):
```go
	// Colorize tints s with the given canonical color; a no-op when color is "".
	Colorize func(s, color string) string
```
Update `DefaultTheme`:
```go
func DefaultTheme() Theme {
	faint := lipgloss.NewStyle().Faint(true)
	return Theme{
		Dim: func(s string) string { return faint.Render(s) },
		Colorize: func(s, color string) string {
			if color == "" {
				return s
			}
			return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(s)
		},
	}
}
```
Update `PlainTheme`:
```go
func PlainTheme() Theme {
	return Theme{
		Dim:      func(s string) string { return s },
		Colorize: func(s, _ string) string { return s },
	}
}
```

- [ ] **Step 4: Add the `color` parameter to `FromResult`**

In `internal/widget/widget.go`, change the signature and set the field:
```go
func FromResult(title, color, lastGood string, res runner.Result) (View, string) {
	v := View{Title: title, Color: color}
```
(Leave the rest of the function body unchanged.)

- [ ] **Step 5: Colorize the border in `Render`**

In `internal/widget/widget.go`, replace the body-assembly section of `Render` (from `top := topBorder(...)` through the final `return b.String()`) with:
```go
	top := theme.Colorize(topBorder(v.Title, w), v.Color)
	bottom := theme.Colorize("└"+strings.Repeat("─", innerW)+"┘", v.Color)
	bar := theme.Colorize("│", v.Color)
	body := buildBody(v, innerW, innerH, theme)

	var b strings.Builder
	b.WriteString(top)
	for _, ln := range body {
		b.WriteByte('\n')
		b.WriteString(bar)
		b.WriteString(ln)
		b.WriteString(bar)
	}
	b.WriteByte('\n')
	b.WriteString(bottom)
	return b.String()
```

- [ ] **Step 6: Update existing widget tests for the no-color baseline**

The existing `TestRenderOK`/`TestRenderPending`/etc. use `PlainTheme()` and `View` literals without `Color`, so they still pass unchanged. Verify any direct `FromResult` callers in `widget_test.go` (if present) pass a color argument; search and fix:

Run: `grep -n "FromResult" internal/widget/widget_test.go`
If any call exists, insert `""` as the second argument: `FromResult("t", "", "", res)`.

- [ ] **Step 7: Update `tui.go` callers**

In `internal/tui/tui.go`, in `New`, set the initial color on the pending view:
```go
			states[r][c] = &widgetState{
				view: widget.View{Title: w.Title, State: widget.StatePending, Color: w.Color},
			}
```
In `Update`'s `widgetResultMsg` case, pass the widget color:
```go
		view, good := widget.FromResult(w.Title, w.Color, st.lastGood, msg.result)
```

- [ ] **Step 8: Update `printer.go` caller**

In `internal/printer/printer.go`, in the views-building loop:
```go
			v, _ := widget.FromResult(w.Title, w.Color, "", results[ri][wi])
```

- [ ] **Step 9: Run the full build and test suite**

Run: `go build ./... && go test ./...`
Expected: build succeeds; all packages PASS.

- [ ] **Step 10: Commit**

```bash
git add internal/widget/widget.go internal/widget/widget_test.go internal/tui/tui.go internal/printer/printer.go
git commit -m "feat(widget): colorize pane border and title"
```

---

### Task 3: Sample configs and documentation

**Files:**
- Modify: `internal/config/default.yaml`
- Modify: `examples/lopanes.yaml`
- Modify: `README.md`

**Interfaces:**
- Consumes: the `color` / `default_color` fields from Task 1. No new code.

- [ ] **Step 1: Add color to the embedded starter config**

In `internal/config/default.yaml`, add `default_color: gray` as the first line, and add a `color` to one widget for illustration. Result:
```yaml
default_interval: 2s
default_timeout: 5s
default_color: gray
rows:
  - height: 1fr
    widgets:
      - {name: clock, title: "Clock", script: "date '+%H:%M:%S'", interval: 1s, color: cyan}
      - {name: uptime, title: "Uptime", script: "uptime"}
  - height: 2fr
    widgets:
      - {name: disk, title: "Disk", script: "df -h / 2>/dev/null | head -n 5", width: 2fr, color: green}
      - {name: mem, title: "Memory", script: "free -h 2>/dev/null || vm_stat"}
```

- [ ] **Step 2: Verify the embedded config still parses**

Run: `go test ./internal/config/ -run TestDefault -v`
Expected: PASS (the existing `default_test.go` parses the embedded YAML).

- [ ] **Step 3: Add color to the example config**

In `examples/lopanes.yaml`, add `default_color: gray` after the `default_timeout` line, and add `color: cyan` to the `clock` widget and `color: green` to the `disk` widget (insert as a new indented `color:` line under each, matching the existing multi-line style).

- [ ] **Step 4: Document the feature in the README**

In `README.md`, add a subsection under the configuration docs:
```markdown
### Pane color

Each pane's border and title can be colored to emphasize it. Set `color` on a
widget, or `default_color` at the top level to color every pane (defaults to
`white`).

```yaml
default_color: gray        # fallback for all panes
rows:
  - widgets:
      - {name: cpu, color: cyan, script: ...}
      - {name: mem, script: ...}              # inherits default_color
```

Accepted values:

- **Names:** `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`,
  `white`, and `bright-*` variants (`gray`/`grey` = `bright-black`).
- **ANSI-256:** a number `0`–`255`.
- **Hex:** `#rgb` or `#rrggbb`.

The body output is never recolored, and the color stays the same in error
states (the `⚠` indicator marks errors). `--no-color` (print mode) strips it.
```
````

- [ ] **Step 5: Final build, test, and manual smoke check**

Run: `go build ./... && go test ./... && ./lopanes --help 2>/dev/null; go run . examples/lopanes.yaml --print 2>&1 | head -20`
Expected: build and tests PASS; print output shows the framed panes (colored unless `--no-color`).

- [ ] **Step 6: Commit**

```bash
git add internal/config/default.yaml examples/lopanes.yaml README.md
git commit -m "docs: document and sample per-pane color"
```

---

## Self-Review Notes

- **Spec coverage:** schema fields + precedence (Task 1, Steps 5–7); parse/validate names/hex/256 (Task 1); `View.Color` + `Theme.Colorize` + `FromResult` param + border-only render (Task 2); tui/printer wiring (Task 2, Steps 7–8); `--no-color` via `PlainTheme` (Task 2, Step 1 `TestRenderPlainThemeNoColor`); always-on-in-error (no recolor logic added — covered by render staying state-independent); docs/samples (Task 3). All spec sections map to a task.
- **Canonical `white` = `"7"`** used consistently across config default, tests, and docs.
- **Build stays green:** `FromResult`'s signature change (Task 2) updates all three callers (widget tests, tui, printer) within the same task before `go test ./...`.
