# quash-board Phase 3: Interactive TUI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add normal mode — a live Bubble Tea dashboard that refreshes each widget on its own interval, reflows on resize, and quits on `q`/`Ctrl-C` — and wire it into the CLI.

**Architecture:** A Bubble Tea `Model` holds a 2D grid of `widgetState` mirroring `config.Rows/Widgets`. Each widget's script runs asynchronously as a `tea.Cmd` (using the Phase 1 `runner`); when a `resultMsg` arrives the model stores it and schedules the next run via a per-widget `tea.Tick` (`tickMsg`). `View` reuses the Phase 2 `widget.Render` + `widget.Frame`. Scripts never block the event loop, so a slow or hanging widget can't freeze the UI.

**Tech Stack:** Go 1.26, `github.com/charmbracelet/bubbletea`. Depends on Phase 1 (`config`, `runner`, `layout`) and Phase 2 (`widget`, `cli`).

**Prerequisite:** Phases 1 and 2 complete (`quash-board --print` works).

---

### Task 1: Add Bubble Tea dependency

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add the dependency**

```bash
cd /home/locle97/coding/github/quash-board
go get github.com/charmbracelet/bubbletea@latest
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: succeeds.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add bubbletea dependency

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: Model construction and message types

Build the `Model` (a grid of `widgetState` mirroring the config) plus the two internal message types. Test that `New` mirrors the config shape.

**Files:**
- Create: `tui/model.go`
- Test: `tui/model_test.go`

- [ ] **Step 1: Write the failing test**

```go
package tui

import (
	"testing"
	"time"

	"github.com/locle97/quash-board/config"
)

func twoRowConfig() *config.Config {
	return &config.Config{
		DefaultInterval: time.Second,
		DefaultTimeout:  time.Second,
		Rows: []config.Row{
			{Height: config.Size{Weight: 1}, Widgets: []config.Widget{
				{Name: "a", Title: "a", Script: "echo a", Interval: time.Second, Timeout: time.Second, Width: config.Size{Weight: 1}},
				{Name: "b", Title: "b", Script: "echo b", Interval: time.Second, Timeout: time.Second, Width: config.Size{Weight: 1}},
			}},
			{Height: config.Size{Weight: 1}, Widgets: []config.Widget{
				{Name: "c", Title: "c", Script: "echo c", Interval: time.Second, Timeout: time.Second, Width: config.Size{Weight: 1}},
			}},
		},
	}
}

func TestNewMirrorsConfigShape(t *testing.T) {
	m := New(twoRowConfig())
	if len(m.states) != 2 {
		t.Fatalf("got %d rows, want 2", len(m.states))
	}
	if len(m.states[0]) != 2 || len(m.states[1]) != 1 {
		t.Fatalf("row widths = %d,%d want 2,1", len(m.states[0]), len(m.states[1]))
	}
	if m.states[0][1].cfg.Name != "b" {
		t.Errorf("states[0][1].cfg.Name = %q, want b", m.states[0][1].cfg.Name)
	}
	if m.states[0][0].result != nil {
		t.Error("new widget should start with nil result (pending)")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./tui/`
Expected: FAIL — `undefined: New`.

- [ ] **Step 3: Write the model and message types**

```go
package tui

import (
	"github.com/locle97/quash-board/config"
	"github.com/locle97/quash-board/runner"
)

// widgetState is the mutable per-widget state in the grid.
type widgetState struct {
	cfg      config.Widget
	result   *runner.Result // latest run; nil = pending
	lastGood *runner.Result // most recent OK run
}

// resultMsg is delivered when a widget's script run completes.
type resultMsg struct {
	row, col int
	res      runner.Result
}

// tickMsg is delivered when a widget's refresh interval elapses.
type tickMsg struct {
	row, col int
}

// Model is the Bubble Tea model for normal mode.
type Model struct {
	cfg    *config.Config
	states [][]widgetState
	w, h   int
}

// New builds a Model whose state grid mirrors cfg.Rows/Widgets.
func New(cfg *config.Config) Model {
	states := make([][]widgetState, len(cfg.Rows))
	for r, row := range cfg.Rows {
		states[r] = make([]widgetState, len(row.Widgets))
		for c, w := range row.Widgets {
			states[r][c] = widgetState{cfg: w}
		}
	}
	return Model{cfg: cfg, states: states}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./tui/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add tui/model.go tui/model_test.go
git commit -m "feat(tui): add model construction and message types

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: Update logic (state transitions)

`Update` handles resize, quit keys, results (store + schedule next tick), and ticks (run again). Tested by feeding messages directly — no real terminal needed.

**Files:**
- Create: `tui/update.go`
- Test: `tui/update_test.go`

- [ ] **Step 1: Write the failing test**

```go
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/locle97/quash-board/runner"
)

func TestUpdateStoresResult(t *testing.T) {
	m := New(twoRowConfig())
	updated, cmd := m.Update(resultMsg{row: 0, col: 1, res: runner.Result{Stdout: "hi"}})
	m = updated.(Model)
	if m.states[0][1].result == nil || m.states[0][1].result.Stdout != "hi" {
		t.Errorf("result not stored: %+v", m.states[0][1].result)
	}
	if m.states[0][1].lastGood == nil {
		t.Error("OK result should update lastGood")
	}
	if cmd == nil {
		t.Error("storing a result should schedule the next tick")
	}
}

func TestUpdateErrorKeepsLastGood(t *testing.T) {
	m := New(twoRowConfig())
	updated, _ := m.Update(resultMsg{row: 0, col: 0, res: runner.Result{Stdout: "good"}})
	m = updated.(Model)
	updated, _ = m.Update(resultMsg{row: 0, col: 0, res: runner.Result{ExitCode: 1, Stderr: "bad"}})
	m = updated.(Model)
	if m.states[0][0].lastGood == nil || m.states[0][0].lastGood.Stdout != "good" {
		t.Errorf("lastGood should be retained, got %+v", m.states[0][0].lastGood)
	}
	if m.states[0][0].result.ExitCode != 1 {
		t.Errorf("result should be the failure, got %+v", m.states[0][0].result)
	}
}

func TestUpdateResizeStoresDimensions(t *testing.T) {
	m := New(twoRowConfig())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updated.(Model)
	if m.w != 100 || m.h != 40 {
		t.Errorf("size = %dx%d, want 100x40", m.w, m.h)
	}
}

func TestUpdateQuitKeys(t *testing.T) {
	m := New(twoRowConfig())
	for _, key := range []string{"q", "ctrl+c"} {
		_, cmd := m.Update(tea.KeyMsg{Type: keyType(key), Runes: keyRunes(key)})
		if cmd == nil {
			t.Errorf("key %q should return a quit command", key)
		}
	}
}

// helpers to build KeyMsg for the two quit keys
func keyType(k string) tea.KeyType {
	if k == "ctrl+c" {
		return tea.KeyCtrlC
	}
	return tea.KeyRunes
}
func keyRunes(k string) []rune {
	if k == "q" {
		return []rune{'q'}
	}
	return nil
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./tui/ -run TestUpdate`
Expected: FAIL — `m.Update` undefined.

- [ ] **Step 3: Write Update and the command helpers**

```go
package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/locle97/quash-board/layout"
	"github.com/locle97/quash-board/runner"
)

// Update handles incoming messages and returns the next model + command.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil

	case resultMsg:
		st := &m.states[msg.row][msg.col]
		res := msg.res
		st.result = &res
		if res.OK() {
			st.lastGood = &res
		}
		return m, m.tickCmd(msg.row, msg.col)

	case tickMsg:
		return m, m.runCmd(msg.row, msg.col)
	}
	return m, nil
}

// runCmd runs one widget's script asynchronously and reports a resultMsg.
func (m Model) runCmd(row, col int) tea.Cmd {
	w := m.states[row][col].cfg
	rect := m.rectFor(row, col)
	env := runner.Env{
		WidgetW: rect.W - 2,
		WidgetH: rect.H - 2,
		Cols:    m.w,
		Lines:   m.h,
	}
	return func() tea.Msg {
		res := runner.Run(context.Background(), w.Script, w.Timeout, env)
		return resultMsg{row: row, col: col, res: res}
	}
}

// tickCmd waits the widget's interval, then emits a tickMsg.
func (m Model) tickCmd(row, col int) tea.Cmd {
	d := m.states[row][col].cfg.Interval
	return tea.Tick(d, func(time.Time) tea.Msg {
		return tickMsg{row: row, col: col}
	})
}

// rectFor returns the current layout rectangle for a widget, or a small
// fallback when the terminal size is not yet known.
func (m Model) rectFor(row, col int) layout.Rect {
	if m.w <= 0 || m.h <= 0 {
		return layout.Rect{W: 20, H: 6}
	}
	rects := layout.Compute(m.cfg, m.w, m.h)
	if row < len(rects) && col < len(rects[row]) {
		return rects[row][col]
	}
	return layout.Rect{W: 20, H: 6}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./tui/ -run TestUpdate`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add tui/update.go tui/update_test.go
git commit -m "feat(tui): handle results, ticks, resize, and quit

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 4: Init and View

`Init` kicks off an immediate run for every widget so boxes populate as soon as scripts return. `View` renders the current state via the Phase 2 `widget` package.

**Files:**
- Create: `tui/view.go`
- Test: `tui/view_test.go`

- [ ] **Step 1: Write the failing test**

```go
package tui

import (
	"strings"
	"testing"

	"github.com/locle97/quash-board/runner"
)

func TestViewBeforeSizeIsEmpty(t *testing.T) {
	m := New(twoRowConfig())
	if m.View() != "" {
		t.Errorf("View before WindowSizeMsg should be empty, got %q", m.View())
	}
}

func TestViewRendersStoredResults(t *testing.T) {
	m := New(twoRowConfig())
	m.w, m.h = 60, 20
	r := runner.Result{Stdout: "ALPHA"}
	m.states[0][0].result = &r
	out := m.View()
	if !strings.Contains(out, "ALPHA") {
		t.Errorf("View should contain widget output:\n%s", out)
	}
	// View should be exactly h lines tall
	if got := strings.Count(out, "\n") + 1; got != 20 {
		t.Errorf("View height = %d lines, want 20", got)
	}
}

func TestInitReturnsCommand(t *testing.T) {
	m := New(twoRowConfig())
	if m.Init() == nil {
		t.Error("Init should return a batch of run commands")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./tui/ -run 'TestView|TestInit'`
Expected: FAIL — `m.View` / `m.Init` undefined.

- [ ] **Step 3: Write Init and View**

```go
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/locle97/quash-board/layout"
	"github.com/locle97/quash-board/widget"
)

// Init dispatches an immediate first run for every widget.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for r := range m.states {
		for c := range m.states[r] {
			cmds = append(cmds, m.runCmd(r, c))
		}
	}
	return tea.Batch(cmds...)
}

// View renders the dashboard at the current terminal size.
func (m Model) View() string {
	if m.w <= 0 || m.h <= 0 {
		return ""
	}
	rects := layout.Compute(m.cfg, m.w, m.h)
	boxes := make([][]string, len(m.states))
	for r := range m.states {
		boxes[r] = make([]string, len(m.states[r]))
		for c := range m.states[r] {
			st := m.states[r][c]
			view := widget.View{
				Widget:   st.cfg,
				Result:   st.result,
				LastGood: st.lastGood,
			}
			boxes[r][c] = widget.Render(view, rects[r][c], false)
		}
	}
	return widget.Frame(rects, boxes)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./tui/`
Expected: PASS (all tui tests).

- [ ] **Step 5: Commit**

```bash
git add tui/view.go tui/view_test.go
git commit -m "feat(tui): add Init dispatch and View rendering

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 5: Run helper and CLI wiring

Add `tui.Run` (starts the program with the alternate screen) and replace the Phase 2 stub in `main.go` so the no-`--print` path launches the live dashboard.

**Files:**
- Create: `tui/run.go`
- Modify: `main.go`

- [ ] **Step 1: Write tui/run.go**

```go
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/locle97/quash-board/config"
)

// Run launches the interactive dashboard and blocks until the user quits.
func Run(cfg *config.Config) error {
	p := tea.NewProgram(New(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
```

- [ ] **Step 2: Replace the stub in main.go**

In `main.go`, change the import block to add the `tui` package:

```go
import (
	"flag"
	"fmt"
	"os"

	"github.com/locle97/quash-board/cli"
	"github.com/locle97/quash-board/config"
	"github.com/locle97/quash-board/printer"
	"github.com/locle97/quash-board/tui"
)
```

Replace these lines:

```go
	// Normal interactive mode is implemented in Phase 3.
	fmt.Fprintln(os.Stderr, "normal mode not yet implemented (use --print)")
	os.Exit(1)
```

with:

```go
	if err := tui.Run(cfg); err != nil {
		fail(err)
	}
```

- [ ] **Step 3: Verify it builds**

Run: `go build ./... && go vet ./...`
Expected: succeeds, vet clean.

- [ ] **Step 4: Manually verify the live dashboard**

```bash
cat > /tmp/qb-live.yaml <<'EOF'
default_interval: 1s
rows:
  - widgets:
      - {name: clock, script: "date +%T"}
      - {name: load,  script: "uptime"}
  - height: 2fr
    widgets:
      - {name: procs, script: "ps -eo pid,comm --sort=-%cpu | head -n 8"}
EOF
go run . --config /tmp/qb-live.yaml
```
Expected: a three-box dashboard. The clock box updates every second. Resizing the terminal reflows the boxes. Pressing `q` or `Ctrl-C` exits cleanly back to the shell prompt.

- [ ] **Step 5: Commit**

```bash
git add tui/run.go main.go
git commit -m "feat(tui): wire interactive normal mode into the CLI

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 6: Phase 3 verification and README

- [ ] **Step 1: Full suite + vet**

Run: `go vet ./... && go test ./...`
Expected: all PASS, vet clean.

- [ ] **Step 2: Confirm modules tidy**

Run: `go mod tidy && git diff --exit-code go.mod go.sum`
Expected: no changes.

- [ ] **Step 3: Write a short README**

Create `README.md`:

```markdown
# quash-board

A YAML-configured TUI dashboard. Each widget runs a shell command and shows its
output in a bordered box. Responsive grid layout.

## Usage

    quash-board [--config PATH] [--print] [--width N] [--no-color]

- Normal mode (default): live dashboard, each widget refreshes on its own
  interval. Quit with `q` or `Ctrl-C`.
- Print mode (`--print`): runs every widget once, prints the dashboard, exits —
  good for snapshots. Redirect with `quash-board --print > snapshot.txt`.

Config search order when `--config` is omitted: `./quash-board.yaml`, then
`~/.config/quash-board/config.yaml`.

## Config

    default_interval: 5s
    default_timeout: 10s
    rows:
      - height: 1fr            # weight (Nfr) or fixed line count
        widgets:
          - name: cpu          # box title unless `title` is set
            script: "top -bn1 | head -n 5"
            interval: 2s        # optional, else default_interval
            timeout: 3s         # optional, else default_timeout
            width: 1fr          # optional weight within the row
          - {name: mem, script: "free -h"}
      - height: 2fr
        widgets:
          - {name: logs, script: "tail -n 20 /var/log/syslog"}

Scripts run via `bash -c` with `WIDGET_W`, `WIDGET_H`, `COLUMNS`, `LINES` set in
the environment. stdout (with ANSI colors) becomes the box body; a non-zero exit
or timeout shows an error state.
```

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: add README with usage and config reference

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

**Phase 3 deliverable:** the complete `quash-board` — both live interactive normal mode and one-shot print mode, configured from YAML, with per-widget intervals, timeouts, error states, and responsive grid layout.
