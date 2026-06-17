# quash-board Phase 2: Rendering + Print Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render widgets into bordered boxes, compose them into a full frame, and ship a working `quash-board --print` command that runs every script once and writes the dashboard to stdout.

**Architecture:** The `widget` package renders one `Result` into a fixed-size bordered box (title embedded in the top border, ANSI-aware truncation/padding, error/pending states) and composes a grid of boxes into a frame string. The `printer` package runs every script once concurrently and produces the frame. `main` parses CLI flags, resolves the config path, detects terminal size, and dispatches to print mode (normal mode is stubbed until Phase 3).

**Tech Stack:** Go 1.26, `github.com/muesli/reflow` (ANSI-aware truncate + width), `golang.org/x/term` (TTY detection + size). Depends on Phase 1 packages (`config`, `runner`, `layout`).

**Prerequisite:** Phase 1 complete (`config`, `runner`, `layout` packages exist and pass tests).

---

### Task 1: Add rendering dependencies

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add the dependencies**

```bash
cd /home/locle97/coding/github/quash-board
go get github.com/muesli/reflow@latest
go get golang.org/x/term@latest
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: succeeds.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add reflow and x/term dependencies

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: Line fitting helper (ANSI-aware truncate + pad)

`fitLine` truncates a string to a visible width (preserving ANSI color sequences) and pads it with spaces to exactly that width. `stripANSI` removes color sequences for `--no-color`.

**Files:**
- Create: `widget/text.go`
- Test: `widget/text_test.go`

- [ ] **Step 1: Write the failing test**

```go
package widget

import "testing"

func TestFitLinePads(t *testing.T) {
	got := fitLine("hi", 5)
	if got != "hi   " {
		t.Errorf("fitLine = %q, want %q", got, "hi   ")
	}
}

func TestFitLineTruncates(t *testing.T) {
	got := fitLine("hello world", 5)
	if visibleWidth(got) != 5 {
		t.Errorf("visible width = %d, want 5 (got %q)", visibleWidth(got), got)
	}
}

func TestFitLinePreservesAnsiWidth(t *testing.T) {
	// colored "hi" should measure as width 2, then pad to 5
	colored := "\x1b[32mhi\x1b[0m"
	got := fitLine(colored, 5)
	if visibleWidth(got) != 5 {
		t.Errorf("visible width = %d, want 5", visibleWidth(got))
	}
}

func TestStripANSI(t *testing.T) {
	got := stripANSI("\x1b[32mhi\x1b[0m")
	if got != "hi" {
		t.Errorf("stripANSI = %q, want hi", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./widget/`
Expected: FAIL — `undefined: fitLine`, `undefined: visibleWidth`, `undefined: stripANSI`.

- [ ] **Step 3: Write the helpers**

```go
package widget

import (
	"regexp"
	"strings"

	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/truncate"
)

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

// visibleWidth returns the printable cell width of s, ignoring ANSI sequences.
func visibleWidth(s string) int { return ansi.PrintableRuneWidth(s) }

// stripANSI removes SGR color/style escape sequences.
func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

// fitLine truncates s to width w (preserving ANSI) and right-pads to exactly w.
func fitLine(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = truncate.String(s, uint(w))
	if pad := w - visibleWidth(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./widget/ -run 'TestFit|TestStrip'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add widget/text.go widget/text_test.go
git commit -m "feat(widget): add ANSI-aware line fitting helpers

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: Widget box rendering

`Render` turns a `View` (the widget config plus its latest/last-good results) into a bordered box of exactly `rect.H` lines, each `rect.W` wide, with the title in the top border and pending/success/error states.

**Files:**
- Create: `widget/widget.go`
- Test: `widget/widget_test.go`

- [ ] **Step 1: Write the failing test**

```go
package widget

import (
	"strings"
	"testing"

	"github.com/locle97/quash-board/config"
	"github.com/locle97/quash-board/layout"
	"github.com/locle97/quash-board/runner"
)

func lines(s string) []string { return strings.Split(s, "\n") }

func TestRenderDimensions(t *testing.T) {
	v := View{Widget: config.Widget{Title: "cpu"}, Result: &runner.Result{Stdout: "23%"}}
	out := Render(v, layout.Rect{W: 12, H: 4}, false)
	ls := lines(out)
	if len(ls) != 4 {
		t.Fatalf("got %d lines, want 4:\n%s", len(ls), out)
	}
	for i, ln := range ls {
		if visibleWidth(ln) != 12 {
			t.Errorf("line %d width = %d, want 12 (%q)", i, visibleWidth(ln), ln)
		}
	}
}

func TestRenderTitleInBorder(t *testing.T) {
	v := View{Widget: config.Widget{Title: "cpu"}, Result: &runner.Result{Stdout: "ok"}}
	out := Render(v, layout.Rect{W: 12, H: 3}, false)
	top := lines(out)[0]
	if !strings.HasPrefix(top, "┌ cpu ") {
		t.Errorf("top border = %q, want to start with '┌ cpu '", top)
	}
}

func TestRenderBody(t *testing.T) {
	v := View{Widget: config.Widget{Title: "cpu"}, Result: &runner.Result{Stdout: "23%"}}
	out := Render(v, layout.Rect{W: 12, H: 3}, false)
	body := stripANSI(lines(out)[1])
	if !strings.Contains(body, "23%") {
		t.Errorf("body = %q, want to contain 23%%", body)
	}
}

func TestRenderPending(t *testing.T) {
	v := View{Widget: config.Widget{Title: "cpu"}} // Result nil
	out := Render(v, layout.Rect{W: 12, H: 3}, false)
	if !strings.Contains(out, "…") {
		t.Errorf("pending box should contain '…':\n%s", out)
	}
}

func TestRenderErrorBanner(t *testing.T) {
	v := View{
		Widget: config.Widget{Title: "db"},
		Result: &runner.Result{ExitCode: 1, Stderr: "boom"},
	}
	out := stripANSI(Render(v, layout.Rect{W: 16, H: 4}, false))
	if !strings.Contains(out, "⚠ exit 1") {
		t.Errorf("want error banner '⚠ exit 1':\n%s", out)
	}
	if !strings.Contains(out, "boom") {
		t.Errorf("want stderr tail 'boom':\n%s", out)
	}
}

func TestRenderTimeoutBanner(t *testing.T) {
	v := View{Widget: config.Widget{Title: "db"}, Result: &runner.Result{TimedOut: true, ExitCode: -1}}
	out := stripANSI(Render(v, layout.Rect{W: 16, H: 4}, false))
	if !strings.Contains(out, "⚠ timed out") {
		t.Errorf("want '⚠ timed out':\n%s", out)
	}
}

func TestRenderErrorFallsBackToLastGood(t *testing.T) {
	v := View{
		Widget:   config.Widget{Title: "db"},
		Result:   &runner.Result{ExitCode: 1, Stderr: "boom"},
		LastGood: &runner.Result{Stdout: "rows: 42"},
	}
	out := stripANSI(Render(v, layout.Rect{W: 16, H: 5}, false))
	if !strings.Contains(out, "rows: 42") {
		t.Errorf("want last-good output 'rows: 42':\n%s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./widget/ -run TestRender`
Expected: FAIL — `undefined: View`, `undefined: Render`.

- [ ] **Step 3: Write the renderer**

```go
package widget

import (
	"fmt"
	"strings"

	"github.com/locle97/quash-board/config"
	"github.com/locle97/quash-board/layout"
	"github.com/locle97/quash-board/runner"
)

// View is everything needed to render one widget box.
type View struct {
	Widget   config.Widget
	Result   *runner.Result // latest run; nil means pending (never run)
	LastGood *runner.Result // most recent OK run, shown dimmed under an error
}

// Render draws the widget into a bordered box of exactly rect.H lines, each
// rect.W cells wide. ANSI colors in the body are preserved unless noColor.
func Render(v View, rect layout.Rect, noColor bool) string {
	innerW := rect.W - 2
	innerH := rect.H - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	body := contentLines(v, noColor)
	if len(body) > innerH {
		body = body[:innerH]
	}
	for len(body) < innerH {
		body = append(body, "")
	}
	for i := range body {
		body[i] = fitLine(body[i], innerW)
	}
	return box(v.Widget.Title, body, rect.W)
}

// contentLines produces the un-fitted body lines for the widget's current state.
func contentLines(v View, noColor bool) []string {
	var banner, src string
	dim := false

	switch {
	case v.Result == nil:
		src = "…"
	case v.Result.OK():
		src = v.Result.Stdout
	default:
		if v.Result.TimedOut {
			banner = "⚠ timed out"
		} else {
			banner = fmt.Sprintf("⚠ exit %d", v.Result.ExitCode)
		}
		if v.LastGood != nil {
			src = v.LastGood.Stdout
			dim = true
		} else {
			src = v.Result.Stderr
		}
	}

	if noColor {
		src = stripANSI(src)
	}
	src = strings.TrimRight(src, "\n")

	var out []string
	if banner != "" {
		out = append(out, banner)
	}
	if src != "" {
		for _, ln := range strings.Split(src, "\n") {
			if dim && !noColor {
				ln = "\x1b[2m" + ln + "\x1b[22m"
			}
			out = append(out, ln)
		}
	}
	return out
}

// box wraps pre-fitted body lines (each innerW wide) in a border with the title
// embedded in the top edge. The result is rect.W wide and len(body)+2 tall.
func box(title string, body []string, w int) string {
	inner := w - 2
	if inner < 0 {
		inner = 0
	}
	label := " " + title + " "
	if visibleWidth(label) > inner {
		label = truncateLabel(label, inner)
	}
	dashes := inner - visibleWidth(label)
	if dashes < 0 {
		dashes = 0
	}

	var b strings.Builder
	b.WriteString("┌" + label + strings.Repeat("─", dashes) + "┐")
	for _, ln := range body {
		b.WriteString("\n│" + ln + "│")
	}
	b.WriteString("\n└" + strings.Repeat("─", inner) + "┘")
	return b.String()
}

func truncateLabel(s string, w int) string { return fitLine(s, w) }
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./widget/ -run TestRender`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add widget/widget.go widget/widget_test.go
git commit -m "feat(widget): render result into bordered box with states

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 4: Frame composition

`Frame` tiles the per-widget box strings into one screen-sized string, row band by row band. Each box is exactly its rect's width and height, so bands concatenate cleanly.

**Files:**
- Create: `widget/frame.go`
- Test: `widget/frame_test.go`

- [ ] **Step 1: Write the failing test**

```go
package widget

import (
	"strings"
	"testing"

	"github.com/locle97/quash-board/layout"
)

func TestFrameConcatenatesRowBoxes(t *testing.T) {
	// two side-by-side 2-line boxes in one band
	rects := [][]layout.Rect{{
		{X: 0, Y: 0, W: 3, H: 2},
		{X: 3, Y: 0, W: 3, H: 2},
	}}
	boxes := [][]string{{
		"AAA\naaa",
		"BBB\nbbb",
	}}
	got := Frame(rects, boxes)
	want := "AAABBB\naaabbb"
	if got != want {
		t.Errorf("Frame =\n%q\nwant\n%q", got, want)
	}
}

func TestFrameStacksBands(t *testing.T) {
	rects := [][]layout.Rect{
		{{X: 0, Y: 0, W: 3, H: 1}},
		{{X: 0, Y: 1, W: 3, H: 1}},
	}
	boxes := [][]string{{"top"}, {"bot"}}
	got := Frame(rects, boxes)
	if got != "top\nbot" {
		t.Errorf("Frame = %q, want %q", got, "top\nbot")
	}
	if strings.Count(got, "\n") != 1 {
		t.Errorf("expected exactly 2 lines, got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./widget/ -run TestFrame`
Expected: FAIL — `undefined: Frame`.

- [ ] **Step 3: Write the composer**

```go
package widget

import (
	"strings"

	"github.com/locle97/quash-board/layout"
)

// Frame tiles rendered boxes into one screen string. boxes[r][c] corresponds to
// rects[r][c]; each box must already be rects[r][c].W wide and .H tall.
func Frame(rects [][]layout.Rect, boxes [][]string) string {
	var screen []string
	for r := range rects {
		if len(rects[r]) == 0 {
			continue
		}
		h := rects[r][0].H
		cols := make([][]string, len(boxes[r]))
		for c := range boxes[r] {
			cols[c] = strings.Split(boxes[r][c], "\n")
		}
		for i := 0; i < h; i++ {
			var line strings.Builder
			for c := range cols {
				if i < len(cols[c]) {
					line.WriteString(cols[c][i])
				}
			}
			screen = append(screen, line.String())
		}
	}
	return strings.Join(screen, "\n")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./widget/`
Expected: PASS (all widget tests).

- [ ] **Step 5: Commit**

```bash
git add widget/frame.go widget/frame_test.go
git commit -m "feat(widget): compose boxes into a full frame

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 5: Print mode renderer

`printer.Render` runs every widget's script once concurrently, renders each box, and returns the composed frame.

**Files:**
- Create: `printer/printer.go`
- Test: `printer/printer_test.go`

- [ ] **Step 1: Write the failing test**

```go
package printer

import (
	"strings"
	"testing"
	"time"

	"github.com/locle97/quash-board/config"
)

func testConfig() *config.Config {
	return &config.Config{
		DefaultInterval: time.Second,
		DefaultTimeout:  2 * time.Second,
		Rows: []config.Row{
			{Height: config.Size{Weight: 1}, Widgets: []config.Widget{
				{Name: "a", Title: "a", Script: "echo AAA", Timeout: time.Second, Width: config.Size{Weight: 1}},
				{Name: "b", Title: "b", Script: "echo BBB", Timeout: time.Second, Width: config.Size{Weight: 1}},
			}},
		},
	}
}

func TestRenderRunsAllScripts(t *testing.T) {
	out := Render(testConfig(), 40, 10, true)
	if !strings.Contains(out, "AAA") || !strings.Contains(out, "BBB") {
		t.Errorf("frame missing script output:\n%s", out)
	}
}

func TestRenderShapesToSize(t *testing.T) {
	out := Render(testConfig(), 40, 10, true)
	ls := strings.Split(out, "\n")
	if len(ls) != 10 {
		t.Errorf("got %d lines, want 10", len(ls))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./printer/`
Expected: FAIL — `undefined: Render`.

- [ ] **Step 3: Write the printer**

```go
package printer

import (
	"context"
	"sync"

	"github.com/locle97/quash-board/config"
	"github.com/locle97/quash-board/layout"
	"github.com/locle97/quash-board/runner"
	"github.com/locle97/quash-board/widget"
)

// Render runs every widget's script once (concurrently) and returns the full
// dashboard frame sized to width x height.
func Render(cfg *config.Config, width, height int, noColor bool) string {
	rects := layout.Compute(cfg, width, height)
	boxes := make([][]string, len(cfg.Rows))

	var wg sync.WaitGroup
	for ri, row := range cfg.Rows {
		boxes[ri] = make([]string, len(row.Widgets))
		for ci, w := range row.Widgets {
			wg.Add(1)
			go func(ri, ci int, w config.Widget) {
				defer wg.Done()
				rect := rects[ri][ci]
				env := runner.Env{
					WidgetW: rect.W - 2,
					WidgetH: rect.H - 2,
					Cols:    width,
					Lines:   height,
				}
				res := runner.Run(context.Background(), w.Script, w.Timeout, env)
				view := widget.View{Widget: w, Result: &res}
				if res.OK() {
					view.LastGood = &res
				}
				boxes[ri][ci] = widget.Render(view, rect, noColor)
			}(ri, ci, w)
		}
	}
	wg.Wait()

	return widget.Frame(rects, boxes)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./printer/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add printer/printer.go printer/printer_test.go
git commit -m "feat(printer): render full dashboard once for print mode

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 6: CLI entrypoint and config path resolution

`main` parses flags, resolves the config path (explicit flag, then `./quash-board.yaml`, then `~/.config/quash-board/config.yaml`), detects terminal size, and dispatches to print mode. Normal mode is stubbed until Phase 3.

**Files:**
- Create: `main.go`
- Create: `cli/cli.go`
- Test: `cli/cli_test.go`

- [ ] **Step 1: Write the failing test for path resolution**

```go
package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigPathExplicit(t *testing.T) {
	got, err := ResolveConfigPath("/tmp/my.yaml", nil)
	if err != nil || got != "/tmp/my.yaml" {
		t.Errorf("got (%q, %v), want /tmp/my.yaml", got, err)
	}
}

func TestResolveConfigPathSearchesCandidates(t *testing.T) {
	dir := t.TempDir()
	want := filepath.Join(dir, "quash-board.yaml")
	if err := os.WriteFile(want, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveConfigPath("", []string{
		filepath.Join(dir, "missing.yaml"),
		want,
	})
	if err != nil || got != want {
		t.Errorf("got (%q, %v), want %q", got, err, want)
	}
}

func TestResolveConfigPathNoneFound(t *testing.T) {
	_, err := ResolveConfigPath("", []string{"/no/such/file.yaml"})
	if err == nil {
		t.Error("expected error when no config found")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cli/`
Expected: FAIL — `undefined: ResolveConfigPath`.

- [ ] **Step 3: Write the resolver and size helpers**

```go
package cli

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// ResolveConfigPath returns explicit if non-empty, else the first existing
// candidate. candidates defaults are provided by the caller (DefaultCandidates).
func ResolveConfigPath(explicit string, candidates []string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("no config file found (looked in: %v); use --config", candidates)
}

// DefaultCandidates is the config search order when --config is not given.
func DefaultCandidates() []string {
	cands := []string{"quash-board.yaml"}
	if home, err := os.UserHomeDir(); err == nil {
		cands = append(cands, home+"/.config/quash-board/config.yaml")
	}
	return cands
}

// PrintSize returns the width and height to use for print mode. If widthFlag > 0
// it wins for width; otherwise the terminal size is used, falling back to 80x24
// when stdout is not a TTY.
func PrintSize(widthFlag int) (width, height int) {
	width, height = 80, 24
	if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		width, height = w, h
	}
	if widthFlag > 0 {
		width = widthFlag
	}
	return width, height
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cli/`
Expected: PASS.

- [ ] **Step 5: Write main.go**

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/locle97/quash-board/cli"
	"github.com/locle97/quash-board/config"
	"github.com/locle97/quash-board/printer"
)

func main() {
	cfgPath := flag.String("config", "", "path to config file")
	printMode := flag.Bool("print", false, "render once to stdout and exit")
	width := flag.Int("width", 0, "override render width (print mode)")
	noColor := flag.Bool("no-color", false, "strip ANSI colors (print mode)")
	flag.Parse()

	path, err := cli.ResolveConfigPath(*cfgPath, cli.DefaultCandidates())
	if err != nil {
		fail(err)
	}
	cfg, err := config.LoadFile(path)
	if err != nil {
		fail(err)
	}

	if *printMode {
		w, h := cli.PrintSize(*width)
		fmt.Println(printer.Render(cfg, w, h, *noColor))
		return
	}

	// Normal interactive mode is implemented in Phase 3.
	fmt.Fprintln(os.Stderr, "normal mode not yet implemented (use --print)")
	os.Exit(1)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "quash-board:", err)
	os.Exit(1)
}
```

- [ ] **Step 6: Verify it builds and print mode works end-to-end**

```bash
go build ./...
cat > /tmp/qb.yaml <<'EOF'
rows:
  - widgets:
      - {name: hello, script: "echo hi from quash-board"}
      - {name: date, script: "date +%Y"}
EOF
go run . --config /tmp/qb.yaml --print --width 60
```
Expected: a two-box dashboard frame printed to stdout, with "hi from quash-board" in the first box and the year in the second.

- [ ] **Step 7: Commit**

```bash
git add main.go cli/cli.go cli/cli_test.go go.mod go.sum
git commit -m "feat(cli): add print-mode entrypoint and config resolution

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 7: Phase 2 verification

- [ ] **Step 1: Run vet and full test suite**

Run: `go vet ./... && go test ./...`
Expected: all PASS, vet clean.

- [ ] **Step 2: Verify a failing-script widget renders an error, not a crash**

```bash
cat > /tmp/qb-err.yaml <<'EOF'
rows:
  - widgets:
      - {name: ok, script: "echo good"}
      - {name: broken, script: "echo nope >&2; exit 2"}
EOF
go run . --config /tmp/qb-err.yaml --print --width 60 --no-color
```
Expected: first box shows "good"; second box shows "⚠ exit 2" and "nope". Process exits 0.

**Phase 2 deliverable:** a working `quash-board --print` that renders the full dashboard to stdout, including error states, sized to the terminal. Phase 3 adds the live interactive mode.
