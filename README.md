# lopanes

A terminal dashboard configured from YAML. Each widget runs a shell command and
displays its output in a bordered box. The grid reflows to your terminal size.

## Install

With a Go toolchain (1.25+):

```bash
go install github.com/locle97/lopanes@latest
```

This installs the `lopanes` binary into `$(go env GOPATH)/bin` (or `$GOBIN`
if set), which must be on your `$PATH`. Pin a specific version with, e.g.,
`go install github.com/locle97/lopanes@v0.1.0`.

### Build from source

```bash
git clone https://github.com/locle97/lopanes.git
cd lopanes
go build -o lopanes .
```

## Usage

```
lopanes [--config PATH] [--print] [--width N] [--no-color]
```

- `--config PATH` — config file. Default search order: `./lopanes.yaml`,
  then `~/.config/lopanes/config.yaml`. If no config is found, lopanes writes
  a starter config to `~/.config/lopanes/config.yaml` and runs it, so the
  first run shows a working dashboard. Edit that file to customize.
- `--print` — run every widget once, print the dashboard, and exit (good for
  snapshots). Otherwise runs the interactive TUI.
- `--width N` — print-mode render width (useful when redirecting to a file).
- `--no-color` — strip ANSI escapes in print mode.

In interactive mode, press `q` or `Ctrl-C` to quit.

### Example

```bash
lopanes --config examples/lopanes.yaml
lopanes --print --config examples/lopanes.yaml --width 120 > snapshot.txt
```

## Configuration

```yaml
default_interval: 5s     # fallback per-widget refresh interval
default_timeout: 10s     # fallback per-widget script timeout
default_color: white     # fallback pane border color
rows:
  - height: 1fr          # weight (Nfr) OR a fixed line count (e.g. 10)
    widgets:
      - name: cpu        # box title unless `title` is set
        script: "./scripts/cpu.sh"
        interval: 2s      # optional, else default_interval
        timeout: 3s       # optional, else default_timeout
        width: 1fr        # optional weight within the row; default equal share
        color: cyan       # optional border color, else default_color
      - name: mem
        script: "free -h"
```

Scripts run via `bash -c` and inherit your environment plus `WIDGET_W`,
`WIDGET_H` (the inner box size), `COLUMNS`, and `LINES`. stdout is the box body
(ANSI colors preserved); a non-zero exit or timeout shows an error indicator and
the stderr tail while keeping the last good output dimmed.

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
