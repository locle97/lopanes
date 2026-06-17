# quash-board — YAML-configured TUI Dashboard

**Date:** 2026-06-17
**Status:** Approved design, ready for implementation plan

## Summary

`quash-board` is a terminal dashboard. The user describes a grid of widgets in a
YAML file; each widget runs a user-supplied shell command and displays its
output inside a bordered box. The dashboard is responsive — it reflows to the
terminal size. It has two modes:

- **Normal mode:** an interactive TUI that refreshes each widget on its own
  interval until the user quits.
- **Print mode:** runs every widget once, renders the full dashboard, prints it
  to stdout, and exits — for snapshotting the dashboard into terminal history or
  a file.

Built in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea)
(event loop) and [Lip Gloss](https://github.com/charmbracelet/lipgloss)
(box/border styling).

## Goals

- Show many widgets in a responsive grid configured entirely from YAML.
- Let users supply their own shell commands; display output verbatim with ANSI
  colors preserved.
- Keep the UI responsive regardless of how slow or broken a script is.
- Provide a print mode for static snapshots.

## Non-goals (v1)

- Typed/rich widgets (gauges, tables, charts rendered by the app). Scripts format
  their own output. The config is designed so a `type` field can be added later
  without breaking changes.
- Nested grids (a cell containing its own sub-grid). v1 is a flat list of rows.
- Interactivity beyond quitting — no scrolling, focus, mouse, or drill-down.
- Hot-reloading the config file while running.

## Architecture

### Concurrency model

Bubble Tea runs a single-threaded Model–Update–View loop over a message channel.
Running scripts inline would freeze the UI, so:

- Each widget's script runs **asynchronously** as a `tea.Cmd`: a goroutine
  invokes `exec.CommandContext` with a timeout and sends a `widgetResultMsg`
  back into the update loop when it finishes.
- A per-widget **timer** (`tea.Tick`) schedules the next run at that widget's
  interval. Fast and slow widgets never block each other.
- The model stores the **last known result** per widget, so the view always has
  something to draw, even while a script is mid-run.

This is the idiomatic Bubble Tea pattern. Rejected alternatives: blocking runs
(freeze the UI); a separate scheduler thread writing shared state (manual locking
that fights the framework).

### Packages

Each package has one responsibility and a clear interface so it can be understood
and tested on its own.

| Package  | Responsibility | Depends on |
|----------|----------------|------------|
| `config` | Parse and validate YAML into typed structs; apply defaults. Pure aside from reading the file. | yaml lib |
| `runner` | Execute one command via `bash -c` with timeout and env injection; return a structured result. No Bubble Tea awareness. | os/exec |
| `layout` | Given terminal `w×h` and the grid spec, compute each widget's box rectangle. Pure function. | — |
| `widget` | Render one result into a bordered box (title, body, error state); clip/pad to its rectangle; preserve ANSI. | lipgloss |
| `tui`    | Bubble Tea Model/Update/View for normal mode; wires timers, runner Cmds, layout, widgets. | all above |
| `print`  | One-shot: run every script once concurrently, render the full frame, write to stdout, exit. | all above |
| `main`   | CLI flag parsing and mode dispatch. | — |

## Configuration

### Schema

```yaml
default_interval: 5s        # fallback per-widget refresh interval
default_timeout: 10s        # fallback per-widget script timeout
rows:
  - height: 1fr             # weight (1fr/2fr/...) OR fixed line count (e.g. 10)
    widgets:
      - name: cpu           # box title unless `title` is set
        script: "./scripts/cpu.sh"
        interval: 2s        # optional, else default_interval
        timeout: 3s         # optional, else default_timeout
        width: 1fr          # optional weight within the row; default equal
      - name: mem
        script: "free -h"
  - height: 2fr
    widgets:
      - {name: logs, title: "Syslog", script: "tail -n 20 /var/log/syslog"}
```

### Fields

- **`default_interval`**, **`default_timeout`**: Go durations (e.g. `2s`, `600s`).
  Applied to any widget that omits its own value.
- **`rows`**: ordered list. Each row spans the full width.
  - **`height`**: either a weight (`Nfr`) sharing leftover vertical space with
    other weighted rows, or a fixed positive integer line count. Rows mixing both
    are supported: fixed rows take their lines first, weighted rows split the rest.
  - **`widgets`**: ordered list, laid out left-to-right within the row.
- **Widget fields:**
  - **`name`** (required): identifier and default box title.
  - **`title`** (optional): overrides the box title.
  - **`script`** (required): shell command string, run as `bash -c "<script>"`.
    May be a one-liner, a path to a script, a pipeline, or a multi-line YAML
    block scalar.
  - **`interval`**, **`timeout`** (optional): override the defaults.
  - **`width`** (optional): weight (`Nfr`) sharing the row's width; default is an
    equal share among the row's widgets.

### Layout model

Flat `rows` → `widgets`. Within a row, widget widths come from their weights
(equal by default). Row heights come from row weights and fixed sizes. Rounding
leftover columns/lines are distributed to the earliest cells so the grid always
fills the terminal exactly. No nested grids in v1.

### Script execution

- Run with `bash -c "<script>"`.
- Environment injected: `WIDGET_W`, `WIDGET_H` (the widget's inner box
  dimensions), `COLUMNS`, `LINES` (terminal size). Existing environment is
  inherited.
- stdout is the widget body; ANSI escape sequences are preserved. stderr is
  captured for the error state.

## Data flow (normal mode)

```
startup → load + validate config → for each widget: one immediate run + one timer
                                          │
        ┌─────────────────────────────────┘
        ▼
   tick fires → tea.Cmd runs script async → widgetResultMsg → store result,
        ▲                                                       re-render View,
        └──────────────── schedule next tick ◄──────────────────┘
   resize msg → recompute layout → re-render
   'q' / Ctrl-C → quit
```

A widget's first run is dispatched at startup so boxes populate as soon as each
script returns, rather than after one interval.

## Rendering

- Each widget is a bordered box with a title. The body is the script's stdout,
  clipped to the inner width and height (lines beyond the box are dropped; long
  lines are truncated). ANSI colors are preserved.
- **Error state:** on non-zero exit or timeout, the box shows the last good
  output dimmed, plus an indicator (`⚠ exit N` or `⚠ timed out`) and a short tail
  of stderr. If there is no prior good output, the box shows only the error.
- **Pending state:** before a widget's first result arrives, the box shows a
  brief placeholder (e.g. `…`).

## Error handling

- **Config errors** (unreadable file, invalid YAML, unknown fields, no widgets,
  unparseable durations or sizes): print a clear message to stderr and exit
  non-zero *before* entering the TUI.
- **Script failure** (non-zero exit or timeout): rendered as the widget error
  state above; the dashboard keeps running.
- **Hangs:** cannot freeze the UI — runs are async and the context timeout kills
  the process. The killed widget enters its timed-out error state.

## Print mode

`quash-board --print`:

1. Loads and validates the config.
2. Runs every widget's script once, concurrently, each with its timeout.
3. Renders the full dashboard at the current terminal size (default 80 columns
   if stdout is not a TTY; height grows to fit content).
4. Writes the rendered frame to stdout with ANSI colors, no alternate screen and
   no input handling, then exits 0.

Flags:

- **`--width N`**: override the render width (useful when redirecting to a file).
- **`--no-color`**: strip ANSI escapes for clean text output.

Redirectable, e.g. `quash-board --print --width 120 > snapshot.txt`.

## CLI

```
quash-board [--config PATH] [--print] [--width N] [--no-color]
```

- **`--config PATH`**: config file path. Default search order: `./quash-board.yaml`,
  then `~/.config/quash-board/config.yaml`.
- **`--print`**: run in print mode (otherwise normal interactive mode).
- **`--width N`**, **`--no-color`**: print-mode rendering overrides.

## Testing strategy

- **`config`**: table-driven unit tests for valid configs, default application,
  and each validation error.
- **`layout`**: unit tests for the box-rectangle math — weight distribution,
  fixed vs weighted rows, rounding remainder placement, tiny terminals.
- **`runner`**: tests against fixture commands covering success, non-zero exit,
  timeout, and env-var injection.
- **`widget` / `print`**: golden-file tests on rendered output strings (normal,
  error, pending states; full-frame print output).
- **`tui`**: drive `Update` with synthetic messages (result, tick, resize, quit)
  and assert model state transitions, without a real terminal.

## Open questions

None blocking. Future extensions noted as non-goals: typed widgets, nested
grids, config hot-reload, and richer interactivity (scroll/focus).
